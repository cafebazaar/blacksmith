package cloudconfig // import "github.com/cafebazaar/blacksmith/cloudconfig"

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net"
	"path"
	"strings"
	"text/template"

	"github.com/cafebazaar/blacksmith/datasource"
)

func findFiles(path string) ([]string, error) {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for i := range infos {
		if !infos[i].IsDir() && infos[i].Name()[0] != '.' {
			files = append(files, infos[i].Name())
		}
	}
	return files, nil
}

//FromPath ????
func FromPath(datasource datasource.GeneralDataSource, tmplPath string) (*template.Template, error) {
	files, err := findFiles(tmplPath)
	if err != nil {
		return nil, err
	}

	t := template.New("")
	t.Delims("<<", ">>")
	t.Funcs(map[string]interface{}{
		"V": func(key string) (interface{}, error) {
			return "FUNC PLACEHOLDER", nil
		},
		"S": func(key string, value string) (interface{}, error) {
			return "FUNC PLACEHOLDER", nil
		},
		"D": func(key string) (interface{}, error) {
			return "FUNC PLACEHOLDER", nil
		},
		"VD": func(key string) (interface{}, error) {
			return "FUNC PLACEHOLDER", nil
		},
		"b64": func(text string) interface{} {
			return "FUNC PLACEHOLDER"
		},
		"b64template": func(text string) (interface{}, error) {
			return "FUNC PLACEHOLDER", nil
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

func (ds *cloudConfigDataSource) ExecuteTemplate(templateName string /*, c *ConfigContext*/) (string, error) {
	// rewrite funcs to include context and hold a lock so it doesn't get overwrite
	buf := new(bytes.Buffer)
	ds.templates.Funcs(map[string]interface{}{
		"V": func(key string) (interface{}, error) {
			if strings.HasPrefix(key, "flags.me.") {
				return ds.currentMachine.GetFlag(key[strings.LastIndex(key, ".")+1:])
			}
			return "", nil
		},
		"S": func(key string, value string) (interface{}, error) {
			if strings.HasPrefix(key, "flags.me.") {
				return "", ds.currentMachine.SetFlag(key[strings.LastIndex(key, ".")+1:], value)
			}
			return "", nil
		},
		"VD": func(key string) (interface{}, error) {
			return ds.currentMachine.GetAndDeleteFlag(key)
		},
		"D": func(key string) (interface{}, error) {
			return "", ds.currentMachine.DeleteFlag(key)
		},
		"b64": func(text string) interface{} {
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"b64template": func(templateName string) (interface{}, error) {
			text, err := ds.ExecuteTemplate(templateName)
			if err != nil {
				return nil, err
			}
			return base64.StdEncoding.EncodeToString([]byte(text)), nil
		},
	})
	err := ds.templates.ExecuteTemplate(buf, templateName, nil)
	if err != nil {
		return "", err
	}
	str := buf.String()
	str = strings.Trim(str, "\n")
	return str, err
}

// What the hell is the parameter FIXME
func (ds *cloudConfigDataSource) macCloudConfig(mac string) (string, error) {

	if ds.templates.Lookup("main") == nil {
		return "", nil
	}
	return ds.ExecuteTemplate("main")
}

var bootParamsRepo *bootParamsDataSource

//MacBootParamsGenerator generates boot params requested by http booter from
//the supplied templates, mac address and datasource
//VERY VERY VERY ILLFORMED
//THE REQUESTS SHOULD BE MADE THROUGH THE INTERFACE THAT http.go PROVIDES.
//YOU ABSOLUTELY NEED TO FIXME
func MacBootParamsGenerator(ds datasource.GeneralDataSource, mac string,
	bootParamsTemplates *template.Template) (string, error) {
	bootParamsRepo.executeLock.Lock()
	defer bootParamsRepo.executeLock.Unlock()

	macAddress, _ := net.ParseMAC(mac)
	bootParamsRepo.GeneralDataSource = ds
	bootParamsRepo.templates = bootParamsTemplates
	bootParamsRepo.currentMachine, _ = ds.GetMachine(macAddress)
	return bootParamsRepo.generateConfig()
}

func (ds *bootParamsDataSource) generateConfig() (string, error) {
	if ds.templates.Lookup("main") == nil {
		return "", nil
	}
	return ds.ExecuteTemplate("main")
}

func (ds *bootParamsDataSource) ExecuteTemplate(templateName string /*, c *ConfigContext*/) (string, error) {
	// rewrite funcs to include context and hold a lock so it doesn't get overwrite
	buf := new(bytes.Buffer)
	ds.templates.Funcs(map[string]interface{}{
		"V": func(key string) (interface{}, error) {
			if strings.HasPrefix(key, "flags.me.") {
				return ds.currentMachine.GetFlag(key[strings.LastIndex(key, ".")+1:])
			}
			return "", nil
		},
		"S": func(key string, value string) (interface{}, error) {
			if strings.HasPrefix(key, "flags.me.") {
				return "", ds.currentMachine.SetFlag(key[strings.LastIndex(key, ".")+1:], value)
			}
			return false, nil
		},
		"VD": func(key string) (interface{}, error) {
			return ds.currentMachine.GetAndDeleteFlag(key)
		},
		"D": func(key string) (interface{}, error) {
			return true, ds.currentMachine.DeleteFlag(key)
		},
		"b64": func(text string) interface{} {
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"b64template": func(templateName string) (interface{}, error) {
			text, err := ds.ExecuteTemplate(templateName)
			if err != nil {
				return nil, err
			}
			return base64.StdEncoding.EncodeToString([]byte(text)), nil
		},
	})
	err := ds.templates.ExecuteTemplate(buf, templateName, nil)
	if err != nil {
		return "", err
	}
	str := buf.String()
	str = strings.Trim(str, "\n")
	return str, err
}
