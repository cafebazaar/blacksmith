package cloudconfig // import "github.com/cafebazaar/blacksmith/cloudconfig"

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"text/template"

	"github.com/cafebazaar/blacksmith/datasource"
)

//MacCloudConfig generates a cloud-config file based on the Mac address that is passed in
//Will generate a commented warning at the end of the cloud-config if the node's ip in the http
//request mismatches the one in etcd
//Part of CloudConfigDataSource interace implementation
func (ds *datasource.) MacCloudConfig(mac string) (string, error) {
	//TODO
	return nil, nil
}

type Repo struct {
	templates   *template.Template
	dataSource  datasource.CloudConfigDataSource
	executeLock sync.Mutex
}

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

func FromPath(ds datasource.CloudConfigDataSource, tmplPath string) (*Repo, error) {
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
	return &Repo{
		templates:  t,
		dataSource: ds,
	}, nil
}

func (r *Repo) ExecuteTemplate(templateName string) (string, error) {
	// rewrite funcs to include context and hold a lock so it doesn't get overwrite
	buf := new(bytes.Buffer)
	// m := TODO
	r.templates.Funcs(map[string]interface{}{
		"V": func(key string) (interface{}, error) {
			return r.dataSource.Get(key)
		},
		"S": func(key string, value string) (interface{}, error) {
			return m.SetFlag(key, value)
		},
		"VD": func(key string) (interface{}, error) {
			return m.GetAndDeleteFlag(key)
		},
		"D": func(key string) (interface{}, error) {
			return m.DeleteFlag(key)
		},
		"b64": func(text string) interface{} {
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"b64template": func(templateName string) (interface{}, error) {
			text, err := r.ExecuteTemplate(templateName, c)
			if err != nil {
				return nil, err
			}
			return base64.StdEncoding.EncodeToString([]byte(text)), nil
		},
	})
	err := r.templates.ExecuteTemplate(buf, templateName, nil)
	if err != nil {
		return "", err
	}
	str := buf.String()
	str = strings.Trim(str, "\n")
	return str, err
}

func (r *Repo) GenerateConfig(c *ConfigContext) (string, error) {
	r.executeLock.Lock()
	defer r.executeLock.Unlock()

	if r.templates.Lookup("main") == nil {
		return "", nil
	}
	return r.ExecuteTemplate("main", c)
}
