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

func templateFuncsDefault() map[string]interface{} {

	emptyStringFunc := func(key string) string {
		return ""
	}

	return map[string]interface{}{
		"machine_variable": emptyStringFunc,
		"cluster_variable": emptyStringFunc,
		"array_variable":   emptyStringFunc,
		"b64":              emptyStringFunc,
		"b64template":      emptyStringFunc,
		"render":           emptyStringFunc,
		"variable":         emptyStringFunc,
		"b64file":          emptyStringFunc,
		"pathSplit":        emptyStringFunc,
		"join":             emptyStringFunc,
	}
}

func templateFuncs(rootTemplate *template.Template, ds *datasource.EtcdDataSource, machineInterface datasource.EtcdMachineInterface) map[string]interface{} {
	return map[string]interface{}{
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
			text, err := executeGeneralTemplate(rootTemplate, templateName, ds, machineInterface)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warnf(
					"error while executeTemplate(templateName=%s machine=%s)",
					templateName, machineInterface.Mac().String())
				return ""
			}
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"render": func(templateName string) string {
			text, err := executeGeneralTemplate(rootTemplate, templateName, ds, machineInterface)
			if err != nil {
				log.WithField("where", "templating.executeTemplate").WithError(err).Warnf(
					"error while executeTemplate(templateName=%s machine=%s)",
					templateName, machineInterface.Mac().String())
				return ""
			}
			return text
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
	}
}

func findFiles(pathStr string) ([]string, error) {
	infos, err := ioutil.ReadDir(pathStr)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, info := range infos {
		if !info.IsDir() && info.Name()[0] != '.' {
			files = append(files, info.Name())
		}
	}

	return files, nil
}

func executeGeneralTemplate(rootTemplate *template.Template, templateName string, ds *datasource.EtcdDataSource, machineInterface datasource.EtcdMachineInterface) (string, error) {
	text, err := executeTemplate(rootTemplate, templateName, ds, machineInterface)
	if err != nil {
		log.WithField("where", "templating.executeTemplate").WithError(err).Warnf(
			"error while executeTemplate(templateName=%s machine=%s)",
			templateName, machineInterface.Mac().String())
		tmpl, err := FSString(false, "/files/"+templateName)
		if err != nil {
			log.Info("Ebedded template not found, attempting to use base: " + err.Error())
			return "", err
		}
		text, err = ExecuteTemplateFile(tmpl, ds, machineInterface)
		if err != nil {
			log.Info("Ebedded template can't be rendered, attempting to use base: " + err.Error())
			return "", err
		}
	}
	return text, nil
}

//FromPath creates templates from the files located in the specified path
func templateFromPath(tmplPath string) (*template.Template, error) {
	files, err := findFiles(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("error while trying to list files in%s: %s", tmplPath, err)
	}

	t := template.New("")
	t.Delims("{{", "}}")
	t.Funcs(templateFuncsDefault())

	for i := range files {
		files[i] = path.Join(tmplPath, files[i])
	}

	t, err = t.ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	return t, nil
}

//FromPath creates templates from the files located in the specified path
func templateForFile(tmpl string) (*template.Template, error) {

	t := template.New("")
	t.Delims("{{", "}}")
	t.Funcs(templateFuncsDefault())

	t, err := t.Parse(tmpl)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func ExecuteTemplateFile(tmpl string,
	ds *datasource.EtcdDataSource, machineInterface datasource.EtcdMachineInterface) (string, error) {

	mac := machineInterface.Mac().String()

	t := template.New("")
	t.Delims("{{", "}}")
	t.Funcs(templateFuncs(t, ds, machineInterface))

	buf := new(bytes.Buffer)
	t, err := t.Parse(tmpl)

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
		ds.WebServer(),
		etcdMembers,
		etcdEndpoints,
	}

	t.Execute(buf, &data)

	if err != nil {
		return "", err
	}
	str := buf.String()
	str = strings.Trim(str, "\n")
	return str, nil
}

func executeTemplate(rootTemplate *template.Template, templateName string,
	ds *datasource.EtcdDataSource, machineInterface datasource.EtcdMachineInterface) (string, error) {
	template := rootTemplate.Lookup(templateName)

	if template == nil {
		return "", fmt.Errorf("template with name=%s wasn't found for root=%v",
			templateName, rootTemplate)
	}

	mac := machineInterface.Mac().String()

	buf := new(bytes.Buffer)
	template.Funcs(templateFuncs(rootTemplate, ds, machineInterface))

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
		ds.WebServer(),
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
	ds *datasource.EtcdDataSource, machineInterface datasource.EtcdMachineInterface) (string, error) {

	template, err := templateFromPath(tmplFolder)
	if err != nil {
		return "", fmt.Errorf("error while reading the template with path=%s: %s",
			tmplFolder, err)
	}

	return executeTemplate(template, templateFile, ds, machineInterface)
}
