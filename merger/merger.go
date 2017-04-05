package merger

import (
	"fmt"

	"github.com/coreos/coreos-cloudinit/config"
	yaml "gopkg.in/yaml.v2"
)

type CloudConfig struct {
	CoreOS         CoreOS          `yaml:"coreos,omitempty"`
	Hostname       string          `yaml:"hostname,omitempty"`
	ManageEtcHosts config.EtcHosts `yaml:"manage_etc_hosts,omitempty"`

	SSHAuthorizedKeys []string      `yaml:"ssh_authorized_keys,omitempty"`
	WriteFiles        []config.File `yaml:"write_files,omitempty"`
	Users             []config.User `yaml:"users,omitempty"`
}

type CoreOS struct {
	Etcd      config.Etcd      `yaml:"etcd,omitempty"`
	Etcd2     config.Etcd2     `yaml:"etcd2,omitempty"`
	Flannel   config.Flannel   `yaml:"flannel,omitempty"`
	Fleet     config.Fleet     `yaml:"fleet,omitempty"`
	Locksmith config.Locksmith `yaml:"locksmith,omitempty"`
	OEM       config.OEM       `yaml:"oem,omitempty"`
	Update    config.Update    `yaml:"update,omitempty"`

	Units []config.Unit `yaml:"units,omitempty"`
}

func Merge(baseCC, userCC CloudConfig) (CloudConfig, error) {
	baseCC.CoreOS = mergeCoreOS(baseCC.CoreOS, userCC.CoreOS)
	baseCC.Hostname = mergeHostname(baseCC.Hostname, userCC.Hostname)
	baseCC.WriteFiles = mergeWriteFiles([]config.File(baseCC.WriteFiles), []config.File(userCC.WriteFiles))
	baseCC.Users = mergeUsers([]config.User(baseCC.Users), []config.User(userCC.Users))
	baseCC.ManageEtcHosts = mergeManageEtcHosts(baseCC.ManageEtcHosts, userCC.ManageEtcHosts)
	baseCC.SSHAuthorizedKeys = mergeSSHAuthorizedKeys(baseCC.SSHAuthorizedKeys, userCC.SSHAuthorizedKeys)
	return baseCC, nil
}

func (cc CloudConfig) String() string {
	bytes, err := yaml.Marshal(cc)
	if err != nil {
		return ""
	}

	stringified := string(bytes)
	stringified = fmt.Sprintf("#cloud-config\n%s", stringified)

	return stringified
}
