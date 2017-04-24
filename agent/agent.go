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
	etcd "github.com/coreos/etcd/client"
	"github.com/pkg/errors"
)

// Status represents an agent status
type Status struct {
	Name        string
	Description string
}

// Heartbeat is the heartbeat details reported by the agent
type Heartbeat struct {
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

func watch(ctx context.Context, kapi etcd.KeysAPI, key string, callback func(*etcd.Response, error)) error {
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

// StartHeartbeat starts a loop for sending a heartbeat every second
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
			json.NewEncoder(buf).Encode(Heartbeat{
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
				fmt.Println(err)
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
		err := watch(ctx, kapi, key, func(resp *etcd.Response, err error) {
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
