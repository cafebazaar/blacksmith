package merger_test

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/cafebazaar/blacksmith/merger"
)

func TestOmitEmpty(t *testing.T) {
	// Ensure unset fields are not included in the marshaled string. Here, we set
	// only one field and expect only that field in the marshaled string.
	cc := merger.CloudConfig{Hostname: "host"}
	if got, want := cc.String(), "#cloud-config\nhostname: host\n"; got != want {
		t.Errorf("got=%q want=%q", got, want)
	}
}

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
  - ssh-rsa AAAAB3NzaC1yc2EAAAA... user1@pc1
  - ssh-rsa AAAAB3NzaC1yc2EAAAA... user2@pc2
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
  - ssh-rsa AAAAB3NzaC1yc2EAAAA... user1@pc1
  - ssh-rsa AAAAB3NzaC1yc2EAAAA... user2@pc2
`,
		},
	}
	for _, tt := range tests {
		baseCC := merger.CloudConfig{}
		if err := yaml.Unmarshal([]byte(tt.base), &baseCC); err != nil {
			t.Error(err)
		}
		userCC := merger.CloudConfig{}
		if err := yaml.Unmarshal([]byte(tt.user), &userCC); err != nil {
			t.Error(err)
		}
		wantCC := merger.CloudConfig{}
		if err := yaml.Unmarshal([]byte(tt.want), &wantCC); err != nil {
			t.Error(err)
		}
		merged, err := merger.Merge(baseCC, userCC)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(merged, wantCC) {
			t.Errorf("got:\n%s \nwant:\n%s", merged.String(), wantCC.String())
		}
	}
}
