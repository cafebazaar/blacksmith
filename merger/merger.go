package merger

import "github.com/coreos/coreos-cloudinit/config"

func Merge(baseCC, userCC config.CloudConfig) (config.CloudConfig, error) {
	baseCC.CoreOS = mergeCoreOS(baseCC.CoreOS, userCC.CoreOS)
	baseCC.Hostname = mergeHostname(baseCC.Hostname, userCC.Hostname)
	baseCC.WriteFiles = mergeWriteFiles([]config.File(baseCC.WriteFiles), []config.File(userCC.WriteFiles))
	baseCC.Users = mergeUsers([]config.User(baseCC.Users), []config.User(userCC.Users))
	baseCC.ManageEtcHosts = mergeManageEtcHosts(baseCC.ManageEtcHosts, userCC.ManageEtcHosts)
	baseCC.SSHAuthorizedKeys = mergeSSHAuthorizedKeys(baseCC.SSHAuthorizedKeys, userCC.SSHAuthorizedKeys)
	return baseCC, nil
}
