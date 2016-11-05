package templating // import "github.com/cafebazaar/blacksmith/templating"

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"

	"github.com/cafebazaar/blacksmith/datasource"
)

func templateFuncs(machineInterface datasource.MachineInterface) map[string]interface{} {
	return map[string]interface{}{
		"getVariable": func(key string) string {
			value, err := machineInterface.GetVariable(key, true)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warn(
					"error while GetVariable")
			}
			return value
		},
		"base64Encode": func(text string) string {
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"base64Decode": func(text string) string {
			data, err := base64.StdEncoding.DecodeString(text)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warn(
					"error while base64Decode")
				return ""
			}
			return string(data)
		},
		"machineKeyCertPair": func(pairName, caKeyName string, size int) *KeyCertPair {
			keyCertPairStr, err := machineInterface.GetVariable(pairName, false)
			if err == nil {
				var keyCertPair KeyCertPair
				if err := json.Unmarshal([]byte(keyCertPairStr), &keyCertPair); err != nil {
					log.WithField("where", "templating.generateKeyCertPair").WithError(err).Warn(
						"error while Unmarshal")
					return nil
				}
				return &keyCertPair
			}

			keyCertPair, err := generateKeyCertPair()
			if err != nil {
				log.WithField("where", "templating.generateKeyCertPair").WithError(err).Warn(
					"error while GenerateKey")
				return nil
			}

			keyCertPairBytes, err := json.Marshal(keyCertPair)
			if err != nil {
				log.WithField("where", "templating.generateKeyCertPair").WithError(err).Warn(
					"error while Marshal")
				return nil
			}
			err = machineInterface.SetVariable(pairName, string(keyCertPairBytes))
			if err != nil {
				log.WithField("where", "templating.generateKeyCertPair").WithError(err).Warn(
					"error while SetVariable")
				return nil
			}

			return keyCertPair
		},
	}
}

func findFiles(path string) ([]string, error) {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []string
	for i := range infos {
		if !infos[i].IsDir() && infos[i].Name()[0] != '.' {
			files = append(files, infos[i].Name())
		}
	}
	return files, nil
}

//FromPath creates templates from the files located in the specifed path
func templateFromPath(tmplPath string, templateFuncs map[string]interface{}) (*template.Template, error) {
	files, err := findFiles(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("error while trying to list files in%s: %s", tmplPath, err)
	}

	t := template.New("")
	// {{ and }} are popular delimiters in the industry. To avoid escaping, let's change them!
	t.Delims("<<", ">>")
	t.Funcs(templateFuncs)

	for i := range files {
		files[i] = path.Join(tmplPath, files[i])
	}

	t, err = t.ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func executeTemplate(rootTemplte *template.Template, templateName string,
	ds datasource.DataSource, machineInterface datasource.MachineInterface,
	webServerAddr string) (string, error) {
	template := rootTemplte.Lookup(templateName)

	if template == nil {
		return "", fmt.Errorf("template with name=%s wasn't found for root=%v",
			templateName, rootTemplte)
	}

	etcdMembers, _ := ds.EtcdMembers()
	machine, err := machineInterface.Machine(false, nil)
	if err != nil {
		return "", err
	}
	data := struct {
		Mac           string
		IP            string
		Hostname      string
		Domain        string
		WebServerAddr string
		EtcdEndpoints string
	}{
		machineInterface.Mac().String(),
		machine.IP.String(),
		machineInterface.Hostname(),
		ds.ClusterName(),
		webServerAddr,
		etcdMembers,
	}
	buf := new(bytes.Buffer)
	err = template.ExecuteTemplate(buf, templateName, &data)
	if err != nil {
		return "", err
	}
	str := buf.String()
	str = strings.Trim(str, "\n")
	return str, nil
}

// ExecuteTemplateFolder returns a string compiled from using the files in the
// specified directory, starting from `main` file inside the directory.
func ExecuteTemplateFolder(tmplFolder string,
	ds datasource.DataSource, machineInterface datasource.MachineInterface,
	webServerAddr string) (string, error) {

	funcs := templateFuncs(machineInterface)
	template, err := templateFromPath(tmplFolder, funcs)
	if err != nil {
		return "", fmt.Errorf("error while reading the template with path=%s: %s",
			tmplFolder, err)
	}

	return executeTemplate(template, "main", ds, machineInterface, webServerAddr)
}
