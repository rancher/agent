//+build !windows

package register

const (
	tokenFile  = "/var/lib/rancher/state/.registration_token"
	apiCrtFile = "/var/lib/cattle/etc/cattle/api.crt"
)
