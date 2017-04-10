package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cafebazaar/blacksmith/swagger/client"
	"github.com/cafebazaar/blacksmith/swagger/client/operations"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/coreos-cloudinit/config"
	"github.com/coreos/coreos-cloudinit/datasource"
	"github.com/coreos/coreos-cloudinit/initialize"
	"github.com/coreos/coreos-cloudinit/network"
	etcd "github.com/coreos/etcd/client"
	"github.com/pkg/errors"
)

// Status represents the status of the agent
type Status struct {
	Name        string
	Description string
}

// Agent details
type Agent struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Age     int       `json:"age"`
}

// WatchOptions represents the callback functions for WatchCommand
type WatchOptions struct {
	RebootCallback  func()
	UpdateCallback  func()
	InstallCallback func()
}

var (
	statusStarting         = Status{"Starting", "blacksmith-agent is starting"}
	statusReady            = Status{"Ready", "blacksmith-agent is ready for new commands"}
	statusConnectingToEtcd = Status{"ConnectingToEtcd", "blacksmith-agent is attempting to connect to etcd"}
	statusBeforeReboot     = Status{"RebootStarted", "blacksmith-agent is attempting reboot"}
	statusBeforeUpdate     = Status{"UpdateStarted", "blacksmith-agent is attempting updating"}
	statusBeforeInstall    = Status{"InstallStarted", "blacksmith-agent is attempting installing"}

	startTime     = time.Now()
	currentStatus = statusStarting
)

// Watch if for watching a key and calling callback on change.
func Watch(ctx context.Context, kapi etcd.KeysAPI, key string, callback func(*etcd.Response, error)) error {
	for {
		ctx, cancel := context.WithTimeout(ctx, time.Hour)
		defer cancel()

		watcher := kapi.Watcher(key, nil)
		resp, err := watcher.Next(ctx)
		if err != nil {
			callback(nil, err)
			continue
		}
		callback(resp, nil)
	}
	return nil
}

func tmpFile(name, content string) *os.File {
	tmpfile, err := ioutil.TempFile("", name)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	return tmpfile
}

func newSwaggerClient(server, caServerName, tlsCert, tlsKey, tlsCa string) *client.Salesman {
	var httpClient *http.Client
	var err error
	if tlsCert != "" && tlsKey != "" && tlsCa != "" {
		tlsCertFile := tmpFile("cert", tlsCert).Name()
		tlsKeyFile := tmpFile("cert-key", tlsKey).Name()
		tlsCaFile := tmpFile("ca", tlsCa).Name()

		httpClient, err = httptransport.TLSClient(httptransport.TLSClientOptions{
			ServerName:         caServerName,
			Certificate:        tlsCertFile,
			Key:                tlsKeyFile,
			CA:                 tlsCaFile,
			InsecureSkipVerify: false,
		})
		if err != nil {
			log.Fatal("Error creating TLSClient:", err)
		}
	} else {
		logrus.Fatal("tlsCert tlsKey tlsCert shoud not be empty")
		httpClient = &http.Client{}
	}

	httpClient.Timeout = time.Second

	transport := httptransport.NewWithClient(
		server,
		client.DefaultBasePath,
		client.DefaultSchemes,
		httpClient,
	)
	return client.New(transport, strfmt.NewFormats())
}

// StartHeartbeat starts a loop for sending heartbeat every second
func StartHeartbeat(ctx context.Context, server, mac, caServerName, tlsCert, tlsKey, tlsCa string) {
	c := newSwaggerClient(server, caServerName, tlsCert, tlsKey, tlsCa)

	canceled := false

	for {
		select {
		case <-ctx.Done():
			canceled = true
		default:
			// Set current status
			buf := new(bytes.Buffer)
			json.NewEncoder(buf).Encode(Agent{
				Status:  currentStatus.Name,
				Message: currentStatus.Description,
				Time:    time.Now(),
				Age:     int(time.Since(startTime) / time.Second),
			})

			ctxReq, _ := context.WithTimeout(ctx, time.Second)
			resp, err := c.Operations.PostHeartbeatMacHeartbeat(&operations.PostHeartbeatMacHeartbeatParams{
				Context:   ctxReq,
				Mac:       mac,
				Heartbeat: buf.String(),
			})
			if err != nil {
				fmt.Printf("Error: err=%#v resp=%v\n", err, resp)
				break
			}

			if err != nil {
				select {
				case <-ctxReq.Done():
					switch ctxReq.Err() {
					case context.Canceled:
						logrus.Info("heartbeat canceled")
					case context.DeadlineExceeded:
						logrus.Error("heartbeat deadline exceeded")
					}
				default:
					logrus.Error(errors.Wrap(err, "heartbeat failed"))
				}
				break
			}

			logrus.WithFields(logrus.Fields{
				"response": resp,
				"status":   currentStatus.Name,
			}).Debug("sent heartbeat")

			/*
				if resp.StatusCode != http.StatusOK {
					buf, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						logrus.WithFields(logrus.Fields{
							"err":    err,
							"status": resp.Status,
						}).Error("heartbeat received non-200 response")
						return
					}
					logrus.WithFields(logrus.Fields{
						"status":   resp.Status,
						"response": string(buf),
					}).Error("heartbeat received non-200 response")
				}
			*/
		}

		if canceled {
			break
		}

		time.Sleep(time.Second)
	}
}

// WatchCommand calls callbacks provided by opts when key is updated
func WatchCommand(ctx context.Context, kapi etcd.KeysAPI, key string, opts WatchOptions) {
	if _, err := kapi.Delete(context.Background(), key, &etcd.DeleteOptions{}); err != nil && !etcd.IsKeyNotFound(err) {
		logrus.WithFields(logrus.Fields{
			"key": key,
		}).Error(errors.Wrap(err, "failed to delete key"))
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		currentStatus = statusReady
		err := Watch(ctx, kapi, key, func(resp *etcd.Response, err error) {
			defer func() {
				currentStatus = statusReady
			}()

			if err != nil {
				currentStatus = statusConnectingToEtcd
				logrus.Error(errors.Wrap(err, "etcd watch"))
				time.Sleep(time.Second)
				return
			}

			logrus.WithFields(logrus.Fields{
				"key":   key,
				"value": resp.Node.Value,
			}).Info("received command")

			switch resp.Node.Value {
			case "reboot":
				currentStatus = statusBeforeReboot
				if opts.RebootCallback != nil {
					opts.RebootCallback()
				}
			case "install":
				currentStatus = statusBeforeInstall
				if opts.InstallCallback != nil {
					opts.InstallCallback()
				}
			case "update":
				currentStatus = statusBeforeUpdate
				if opts.UpdateCallback != nil {
					opts.UpdateCallback()
				}
			default:
				logrus.WithFields(logrus.Fields{
					"command": resp.Node.Value,
				}).Error("unknown command")
			}
		})

		if err != nil {
			currentStatus = statusConnectingToEtcd
			logrus.Error(errors.Wrap(err, "etcd watch failed"))
			time.Sleep(time.Second)
		}
	}
}

func GetCloudConfig(url string) ([]byte, error) {
	httpClient := http.Client{
		Timeout: time.Second,
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "cloudconfig fetch failed")
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadAll for cloudconfig response failed")
	}
	return buf, nil
}

func ApplyCloudconfig(cloudconfig []byte) error {
	var (
		configRoot     = ""
		flagWorkspace  = "/var/lib/coreos-cloudinit"
		flagSshKeyName = "coreos-cloudinit"
	)

	var ccu *config.CloudConfig
	switch ud, err := initialize.ParseUserData(string(cloudconfig)); err {
	case initialize.ErrIgnitionConfig:
		return errors.Wrap(err, "Detected an Ignition config")
	case nil:
		switch t := ud.(type) {
		case *config.CloudConfig:
			ccu = t
		default:
			return fmt.Errorf("Only CloudConfig user-data is supported")
		}
	default:
		return errors.Wrap(err, "Failed to parse user-data")
	}

	env := initialize.NewEnvironment("/", configRoot, flagWorkspace, flagSshKeyName, datasource.Metadata{})
	if err := initialize.Apply(*ccu, []network.InterfaceGenerator{}, env); err != nil {
		return errors.Wrap(err, "Failed to apply cloud-config")
	}

	return nil
}
