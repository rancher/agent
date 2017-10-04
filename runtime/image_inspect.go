package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utils"
	v3 "github.com/rancher/go-rancher/v3"
)

const (
	wwwAuthenticateHeader = "Www-Authenticate"
)

// ImageInspect calls registry API to get the specification of an image without pulling it
func ImageInspect(imageName string, credential v3.Credential) (map[string]interface{}, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsconfig.ClientDefault(),
		},
	}
	// parse the image to get repo, tag or digest
	named, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return nil, err
	}
	serverAddress := reference.Domain(named)
	if serverAddress == "docker.io" {
		serverAddress = "index.docker.io"
	}
	repository := ""
	tag := ""
	digest := ""
	repository = strings.Split(reference.Path(named), ":")[0]
	if tagged, ok := named.(reference.Tagged); ok {
		tag = tagged.Tag()
	}
	if digested, ok := named.(reference.Digested); ok {
		digest = digested.String()
	}
	if tag == "" {
		tag = "latest"
	}

	// construct v2 Registry API
	v2Base, err := url.Parse(serverAddress)
	if err != nil {
		return nil, err
	}
	if v2Base.Scheme == "" {
		v2Base.Scheme = "https"
	}
	v2Base.Path = path.Join(v2Base.Path, "v2")

	// contact v2 registry first to determine where we should authenticate with registry
	response, err := httpClient.Get(v2Base.String())
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusOK {
		// registry implemented v2 API and doesn't require auth
		if digest == "" {
			digest, err = getDigest(v2Base, repository, tag, "", "", httpClient)
			if err != nil {
				return nil, err
			}
		}
		if digest == "" {
			return nil, errors.Errorf("can't find digest for %s", imageName)
		}
		// get blob info by referencing content digest
		return getBlobData(v2Base, repository, digest, "", "", httpClient)
	} else if response.StatusCode == http.StatusNotFound {
		// registry doesn't implement v2 API
		return nil, errors.Errorf("the registry %s doesn't implement Docker v2 API", serverAddress)
	} else if response.StatusCode == http.StatusUnauthorized {
		// if it returns 401 then we look for Www-Authenticate header

		// get token from Www-Authenticate header
		authHeader := response.Header.Get(wwwAuthenticateHeader)
		authSchema, authURL := parseAuthHeader(authHeader)
		if authSchema == "" || authURL == nil {
			return nil, errors.Errorf("failed to parse Www-Authenticate header %s", authHeader)
		}
		query := authURL.Query()
		query.Add("scope", fmt.Sprintf("repository:%s:pull", repository))
		authURL.RawQuery = query.Encode()
		request, err := http.NewRequest(http.MethodGet, authURL.String(), nil)
		if err != nil {
			return nil, err
		}

		basicAuth := ""
		if credential.PublicValue != "" {
			request.SetBasicAuth(credential.PublicValue, credential.SecretValue)
			auth := credential.PublicValue + ":" + credential.SecretValue
			basicAuth = base64.StdEncoding.EncodeToString([]byte(auth))
		}

		dataMap, err := callAndMarshallBody(httpClient, request)
		if err != nil {
			return nil, err
		}
		token := ""
		if v, ok := dataMap["token"]; ok {
			token = v.(string)
		}

		// if digest is empty then get digest first by calling get manifest v2
		authToken := ""
		if digest == "" {
			if authSchema == "Bearer" {
				authToken = token
			} else if authSchema == "Basic" {
				authToken = basicAuth
			}
			digest, err = getDigest(v2Base, repository, tag, authSchema, authToken, httpClient)
			if err != nil {
				return nil, err
			}
		}
		if digest == "" {
			return nil, errors.Errorf("can't find digest for %s", imageName)
		}

		// get blob info by referencing content digest
		return getBlobData(v2Base, repository, digest, authSchema, authToken, httpClient)
	}
	return nil, errors.Errorf("Unknown status code %v", response.StatusCode)
}

// parseAuthHeader parse a Www-Authenticate header to return a url and auth schema
// format is usually like Www-Authenticate: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:samalba/my-app:pull,push"
func parseAuthHeader(header string) (string, *url.URL) {
	schema := ""
	authURL := &url.URL{}
	parts := strings.Split(header, ",")
	for _, part := range parts {
		keyPair := strings.SplitN(part, "=", 2)
		if len(keyPair) == 2 {
			if strings.Contains(keyPair[0], "realm") {
				if keyPair[0] == "Bearer realm" {
					schema = "Bearer"
				} else if keyPair[1] == "Basic realm" {
					schema = "Basic"
				}
				u, err := url.Parse(strings.Trim(keyPair[1], "\""))
				if err != nil {
					return "", nil
				}
				authURL = u
			} else {
				q := authURL.Query()
				q.Add(keyPair[0], strings.Trim(keyPair[1], "\""))
				authURL.RawQuery = q.Encode()
			}
		}
	}
	return schema, authURL
}

func getDigest(baseURL *url.URL, repository, tag, authSchema, authToken string, client *http.Client) (string, error) {
	manifestURL := url.URL{}
	manifestURL.Scheme = baseURL.Scheme
	manifestURL.Path = path.Join(baseURL.Path, repository, "manifests", tag)
	request, err := http.NewRequest(http.MethodGet, manifestURL.String(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	if authSchema != "" {
		request.Header.Set("Authorization", fmt.Sprintf("%s %s", authSchema, authToken))
	}
	dataMap, err := callAndMarshallBody(client, request)
	if err != nil {
		return "", err
	}
	if dig, ok := utils.GetFieldsIfExist(dataMap, "config", "digest"); ok {
		return utils.InterfaceToString(dig), nil
	}
	return "", errors.New("Can't find digest from response")
}

func getBlobData(baseURL *url.URL, repository, digest, authSchema, authToken string, client *http.Client) (map[string]interface{}, error) {
	blobURL := url.URL{}
	blobURL.Scheme = baseURL.Scheme
	blobURL.Path = path.Join(baseURL.Path, repository, "blobs", digest)
	request, err := http.NewRequest(http.MethodGet, blobURL.String(), nil)
	if err != nil {
		return nil, err
	}
	if authSchema != "" {
		request.Header.Set("Authorization", fmt.Sprintf("%s %s", authSchema, authToken))
	}
	return callAndMarshallBody(client, request)
}

func callAndMarshallBody(client *http.Client, request *http.Request) (map[string]interface{}, error) {
	reader, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer reader.Body.Close()
	buffer, err := ioutil.ReadAll(reader.Body)
	if err != nil {
		return nil, err
	}
	dataMap := map[string]interface{}{}
	if err := json.Unmarshal(buffer, &dataMap); err != nil {
		return nil, err
	}
	return dataMap, nil
}
