package merger_test

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/cafebazaar/blacksmith/merger"
	"github.com/coreos/coreos-cloudinit/config"
)

func TestMerge(t *testing.T) {
	tests := []struct{ base, user, want string }{
		{
			base: `
#cloud-config
coreos:
  update:
    reboot-strategy: etcd-lock
  units:
    - name: 1
    - name: etcd2.service
      command: start
    - name: docker.service
      drop-ins:
        - name: 50-insecure-registry.conf
          content: |
            [Service]
            Environment=DOCKER_OPTS=--insecure-registry=localhost:5000
write_files:
  - path: /var/lib/blacksmith/workspace-updater.sh
    encoding: "base64"
    permissions: "0774"
    owner: "root"
    content: IyEvdXNyL2...
  - path: /tmp/blacksmith-agent.sh
    encoding: "base64"
    permissions: "0744"
    owner: "root"
    content: IyEvYmluL2...
    `,
			user: `
#cloud-config
coreos:
  update:
    reboot-strategy: etcd-lock
  units:
    # Some comment
    - name: 2
    - name: etcd2.service
      command: start
    - name: docker.service
      drop-ins:
        - name: 50-insecure-registry.conf
          content: |
            [Service]
            Environment=DOCKER_OPTS=--insecure-registry=172.20.0.1:5000
write_files:
  - path: /var/lib/blacksmith/workspace-updater.sh
    encoding: "base64"
    permissions: "0774"
    owner: "root"
    content: IyEvdXNyL2...
  - path: /tmp/blacksmith-bootstrapper.sh
    encoding: "base64"
    permissions: "0744"
    owner: "root"
    content: IyEvYmluL2...
  - path: /var/lib/blacksmith/workspaces/initial.yaml
    encoding: "base64"
    permissions: "0744"
    owner: "root"
    content: Y2x1c3Rlci...
ssh_authorized_keys:
  - ssh-rsa AAAAB3NzaC1yc2EAAAA... sina@cafesina
  - ssh-rsa AAAAB3NzaC1yc2EAAAA... ali@ali-javadi-pc
            `,
			want: `
#cloud-config
coreos:
  update:
    reboot-strategy: etcd-lock
  units:
    # Some comment
    - name: 1
    - name: etcd2.service
      command: start
    - name: docker.service
      drop-ins:
        - name: 50-insecure-registry.conf
          content: |
            [Service]
            Environment=DOCKER_OPTS=--insecure-registry=172.20.0.1:5000
    - name: 2
write_files:
  - path: /var/lib/blacksmith/workspace-updater.sh
    encoding: "base64"
    permissions: "0774"
    owner: "root"
    content: IyEvdXNyL2...
  - path: /tmp/blacksmith-agent.sh
    encoding: "base64"
    permissions: "0744"
    owner: "root"
    content: IyEvYmluL2...
  - path: /tmp/blacksmith-bootstrapper.sh
    encoding: "base64"
    permissions: "0744"
    owner: "root"
    content: IyEvYmluL2...
  - path: /var/lib/blacksmith/workspaces/initial.yaml
    encoding: "base64"
    permissions: "0744"
    owner: "root"
    content: Y2x1c3Rlci...
ssh_authorized_keys:
  - ssh-rsa AAAAB3NzaC1yc2EAAAA... sina@cafesina
  - ssh-rsa AAAAB3NzaC1yc2EAAAA... ali@ali-javadi-pc
`,
		},
	}
	for _, tt := range tests {
		baseCC := config.CloudConfig{}
		if err := yaml.Unmarshal([]byte(tt.base), &baseCC); err != nil {
			t.Error(err)
		}
		userCC := config.CloudConfig{}
		if err := yaml.Unmarshal([]byte(tt.user), &userCC); err != nil {
			t.Error(err)
		}
		wantCC := config.CloudConfig{}
		if err := yaml.Unmarshal([]byte(tt.want), &wantCC); err != nil {
			t.Error(err)
		}
		if got, err := merger.Merge(baseCC, userCC); !reflect.DeepEqual(got, wantCC) {
			if err != nil {
				t.Error(err)
			}
			t.Errorf("got:\n%s \nwant:\n%s", got.String(), wantCC.String())
		}
	}
}
