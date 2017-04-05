package merger

import "github.com/coreos/coreos-cloudinit/config"

func mergeCoreOS(coreos1, coreos2 CoreOS) CoreOS {
	if (Etcd{}) != coreos2.Etcd {
		coreos1.Etcd = coreos2.Etcd
	}
	if (Etcd2{}) != coreos2.Etcd2 {
		coreos1.Etcd2 = coreos2.Etcd2
	}
	if (Flannel{}) != coreos2.Flannel {
		coreos1.Flannel = coreos2.Flannel
	}
	if (Fleet{}) != coreos2.Fleet {
		coreos1.Fleet = coreos2.Fleet
	}
	if (Locksmith{}) != coreos2.Locksmith {
		coreos1.Locksmith = coreos2.Locksmith
	}
	if (OEM{}) != coreos2.OEM {
		coreos1.OEM = coreos2.OEM
	}
	if (Update{}) != coreos2.Update {
		coreos1.Update = coreos2.Update
	}
	coreos1.Units = mergeUnits(
		[]Unit(coreos1.Units),
		[]Unit(coreos2.Units),
	)
	return coreos1
}

func mergeUnits(units1, units2 []Unit) []Unit {
	var units []Unit
	for _, b := range units1 {
		if j := indexOfUnit(b, units2); j != -1 {
			units = append(units, units2[j])
			units2 = append(units2[:j], units2[j+1:]...)
		} else {
			units = append(units, b)
		}
	}
	return append(units, units2...)
}

func mergeHostname(hostname1, hostname2 string) string {
	if hostname2 != "" {
		return hostname2
	}
	return hostname1
}

func mergeWriteFiles(files1, files2 []File) []File {
	var files []File
	for _, b := range ([]File)(files1) {
		if j := indexOfFile(b, files2); j != -1 {
			files = append(files, files2[j])
			files2 = append(files2[:j], files2[j+1:]...)
		} else {
			files = append(files, b)
		}
	}
	return append(files, files2...)
}

func mergeUsers(users1, users2 []User) (result []User) {
	for _, b := range users1 {
		if j := indexOfUser(b, users2); j != -1 {
			result = append(result, users2[j])
			users2 = append(users2[:j], users2[j+1:]...)
		} else {
			result = append(result, b)
		}
	}
	return append(result, users2...)
}

func mergeManageEtcHosts(hosts1, hosts2 config.EtcHosts) config.EtcHosts {
	if "" != hosts2 {
		return hosts2
	}
	return hosts1
}

func mergeSSHAuthorizedKeys(keys1, keys2 []string) []string {
	return append(keys1, keys2...)
}

func indexOfUnit(unit Unit, units []Unit) int {
	for i, u := range units {
		if u.Name == unit.Name {
			return i
		}
	}
	return -1
}

func indexOfFile(file File, files []File) int {
	for i, f := range files {
		if f.Path == file.Path {
			return i
		}
	}
	return -1
}

func indexOfUser(item User, slice []User) int {
	for index, i := range slice {
		if i.Name == item.Name {
			return index
		}
	}
	return -1
}
