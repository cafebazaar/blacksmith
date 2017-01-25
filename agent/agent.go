package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
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
	Status  string `json:"status"`
	Message string `json:"message"`
	Time    string `json:"time"`
	Age     int    `json:"age"`
}

// WatchOptions represents the callback functions for WatchCommand
type WatchOptions struct {
	RebootCallback func()
	UpdateCallback func()
}

var (
	statusStarting         = Status{"Starting", "blacksmith-agent is starting"}
	statusReady            = Status{"Ready", "blacksmith-agent is ready for new commands"}
	statusConnectingToEtcd = Status{"ConnectingToEtcd", "blacksmith-agent is attempting to connect to etcd"}
	statusBeforeReboot     = Status{"BeforeReboot", "blacksmith-agent is attempting reboot"}
	statusBeforeUpdate     = Status{"BeforeUpdate", "blacksmith-agent is attempting updating"}

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

// StartHeartbeat starts a loop for sending heartbeat every second
func StartHeartbeat(ctx context.Context, heartbeatURL *url.URL) {
	httpClient := http.Client{
		Timeout: time.Second,
	}

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
				Time:    time.Now().Format(time.RFC822),
				Age:     int(time.Since(startTime) / time.Second),
			})

			req, err := http.NewRequest("POST", heartbeatURL.String(), buf)
			if err != nil {
				logrus.Error(errors.Wrap(err, "request initialization failed"))
				break
			}

			ctxReq, _ := context.WithTimeout(ctx, time.Second)
			req = req.WithContext(ctxReq)
			resp, err := httpClient.Do(req)

			if err != nil {
				select {
				case <-req.Context().Done():
					switch req.Context().Err() {
					case context.Canceled:
						logrus.Info("heartbeat canceled")
					case context.DeadlineExceeded:
						logrus.Error("heartbeat deadline exceeded")
					}
				default:
					logrus.Error(errors.Wrap(err, "heartbeat request failed"))
				}
				break
			}

			logrus.WithFields(logrus.Fields{
				"url":      heartbeatURL.String(),
				"response": resp.StatusCode,
				"status":   currentStatus.Name,
			}).Debug("sent a heartbeat")

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
