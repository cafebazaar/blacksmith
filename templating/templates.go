package templating // import "github.com/cafebazaar/blacksmith/templating"

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"

	"github.com/cafebazaar/blacksmith/datasource"
)

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
func templateFromPath(tmplPath string) (*template.Template, error) {
	files, err := findFiles(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("error while trying to list files in%s: %s", tmplPath, err)
	}

	t := template.New("")
	t.Delims("<<", ">>")
	t.Funcs(map[string]interface{}{
		"machine_variable": func(key string) string {
			return ""
		},
		"cluster_variable": func(key string) string {
			return ""
		},
		"array_variable": func(key string) string {
			return ""
		},
		"b64": func(text string) string {
			return ""
		},
		"b64template": func(templateName string) string {
			return ""
		},
		"variable": func(key string) string {
			return ""
		},
		"b64file": func(fileName string) string {
			return ""
		},
		"pathSplit": func(pathStr string) string {
			return ""
		},
		"join": func(str string) string {
			return ""
		},
	})

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

	mac := machineInterface.Mac().String()

	buf := new(bytes.Buffer)
	template.Funcs(map[string]interface{}{
		"machine_variable": func(key string) string {
			value, err := machineInterface.GetVariable(key)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warn(
					"error while GetVariable")
			}
			return value
		},
		"cluster_variable": func(key string) string {
			value, err := ds.GetClusterVariable(key)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warn(
					"error while GetVariable")
			}
			return value
		},
		"array_variable": func(key string) interface{} {
			value, err := ds.GetArrayVariable(key)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warn(
					"error while GetVariable")
			}
			return value
		},
		"variable": func(key string) string {
			value, err := ds.GetVariable(key)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warn(
					"error while GetVariable")
			}
			return value
		},
		"b64": func(text string) string {
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"b64template": func(templateName string) string {
			text, err := executeTemplate(rootTemplte, templateName, ds, machineInterface, webServerAddr)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warnf(
					"error while executeTemplate(templateName=%s machine=%s)",
					templateName, mac)
				return ""
			}
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"b64file": func(fileName string) string {
			text, err := ioutil.ReadFile(path.Join(ds.WorkspacePath(), fileName))
			if err != nil {
				return err.Error()
			}
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"pathSplit": func(pathStr string) string {
			_, file := path.Split(pathStr)
			return file
		},
		"join": func(str1 string, str2 string) string {
			return strings.Join([]string{str1, str2}, "")
		},
	})

	etcdMembers, _ := ds.EtcdMembers()
	etcdEndpoints, _ := ds.EtcdEndpoints()

	machine, err := machineInterface.Machine(false, nil)
	if err != nil {
		return "", err
	}

	data := struct {
		Mac              string
		IP               string
		Hostname         string
		Domain           string
		FileServerAddr   string
		WebServerAddr    string
		EtcdEndpoints    string
		EtcdCtlEndpoints string
	}{
		mac,
		machine.IP.String(),
		machineInterface.Hostname(),
		ds.ClusterName(),
		ds.FileServer(),
		webServerAddr,
		etcdMembers,
		etcdEndpoints,
	}
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
func ExecuteTemplateFolder(tmplFolder string, templateFile string,
	ds datasource.DataSource, machineInterface datasource.MachineInterface,
	webServerAddr string) (string, error) {

	template, err := templateFromPath(tmplFolder)
	if err != nil {
		return "", fmt.Errorf("error while reading the template with path=%s: %s",
			tmplFolder, err)
	}

	return executeTemplate(template, templateFile, ds, machineInterface, webServerAddr)
}
