package k8s

import "net/http"

const version = `{
  "major": "1",
  "minor": "2+",
  "gitVersion": "v1.2.0-rancher-1",
  "gitCommit": "ed6532f975cff184196dbe214de8fa0198b415ef",
  "gitTreeState": "clean"
}`

func Version(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	rw.Write([]byte(version))
}
