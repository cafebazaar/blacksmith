package web

import (
	"encoding/json"
	"net"
	"path"
	"strings"
	"time"

	"github.com/cafebazaar/blacksmith/swagger/models"
	"github.com/cafebazaar/blacksmith/swagger/restapi/operations"
	"github.com/cafebazaar/blacksmith/templating"
	"github.com/coreos/etcd/client"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

func (ws *webServer) swaggerPostWorkspaceInstallMacHandler(params operations.PostWorkspaceInstallMacParams) middleware.Responder {
	mac, err := net.ParseMAC(params.Mac)
	if err != nil {
		return operations.
			NewPostWorkspaceInstallMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	if err := ws.ds.GetEtcdMachine(mac).SetVariable(path.Join("agent_command"), "install"); err != nil {
		return operations.
			NewPostWorkspaceInstallMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	return operations.NewPostWorkspaceInstallMacOK()
}

func (ws *webServer) swaggerPostWorkspacesHandler(params operations.PostWorkspacesParams) middleware.Responder {
	if err := ws.ds.UpdateWorkspaces(); err != nil {
		return operations.
			NewPostWorkspacesInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	return operations.NewPostWorkspacesOK()
}

func (ws *webServer) swaggerPostWorkspaceUpdateMacHandler(params operations.PostWorkspaceUpdateMacParams) middleware.Responder {
	mac, err := net.ParseMAC(params.Mac)
	if err != nil {
		return operations.
			NewPostWorkspaceUpdateMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	if err := ws.ds.GetEtcdMachine(mac).SetVariable(path.Join("agent_command"), "update"); err != nil {
		return operations.
			NewPostWorkspaceUpdateMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	return operations.NewPostWorkspaceUpdateMacOK()
}

func (ws *webServer) swaggerPostRebootMacHandler(params operations.PostRebootMacParams) middleware.Responder {
	mac, err := net.ParseMAC(params.Mac)
	if err != nil {
		return operations.
			NewPostRebootMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	if err := ws.ds.GetEtcdMachine(mac).SetVariable(path.Join("agent_command"), "reboot"); err != nil {
		return operations.
			NewPostRebootMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	return operations.NewPostRebootMacOK()
}

func (ws *webServer) swaggerGetCloudconfigCcMacHander(params operations.GetCloudconfigCcMacParams) middleware.Responder {
	config, err := ws.generateTemplateForMachine("cloudconfig", params.Mac)
	if err != nil {
		return operations.
			NewGetCloudconfigCcMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	if config != "" && *params.Validate {
		return operations.
			NewGetCloudconfigCcMacOK().
			WithPayload(models.Cloudconfig(config + templating.ValidateCloudConfig(config)))
	}
	return operations.
		NewGetCloudconfigCcMacOK().
		WithPayload(models.Cloudconfig(config))
}

func (ws *webServer) swaggerGetCloudconfigIgMacHander(params operations.GetCloudconfigIgMacParams) middleware.Responder {
	config, err := ws.generateTemplateForMachine("ignition", params.Mac)
	if err != nil {
		return operations.
			NewGetCloudconfigIgMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	return operations.
		NewGetCloudconfigIgMacOK().
		WithPayload(models.Cloudconfig(config))
}

func (ws *webServer) swaggerGetCloudconfigBpMacHander(params operations.GetCloudconfigBpMacParams) middleware.Responder {
	config, err := ws.generateTemplateForMachine("bootparams", params.Mac)
	if err != nil {
		return operations.
			NewGetCloudconfigBpMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	return operations.
		NewGetCloudconfigBpMacOK().
		WithPayload(models.Cloudconfig(config))
}

func (ws *webServer) swaggerGetNodesHander(params operations.GetNodesParams) middleware.Responder {
	machines, err := ws.ds.GetEtcdMachines()
	if err != nil {
		return operations.
			NewGetNodesInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	resp := operations.NewGetNodesOK()
	for _, machine := range machines {
		m, err := machineToDetails(machine)
		if err != nil {
			return operations.
				NewGetNodesInternalServerError().
				WithPayload(&models.Error{err.Error()})
		}
		if m == nil {
			continue
		}

		value, err := machine.GetVariable(path.Join("agent_heartbeat"))
		if err != nil {
			return operations.
				NewGetNodesInternalServerError().
				WithPayload(&models.Error{err.Error()})
		}
		heartbeat := struct {
			Time   time.Time `json:"time"`
			Status string    `json:"status"`
		}{}
		json.NewDecoder(strings.NewReader(value)).Decode(&heartbeat)
		resp.Payload = append(resp.Payload, &models.Node{
			IP:            m.IP.String(),
			Mac:           machine.Mac().String(),
			LastHeartbeat: strfmt.Date(heartbeat.Time),
			Age:           int64(time.Now().Sub(heartbeat.Time).Seconds()),
		})
	}
	return resp
}

func (ws *webServer) swaggerGetVariablesClusterKeyHandler(params operations.GetVariablesClusterKeyParams) middleware.Responder {
	value, err := ws.ds.GetClusterVariable(params.Key)
	if client.IsKeyNotFound(err) {
		return operations.
			NewGetVariablesClusterKeyNotFound()
	}

	if err != nil {
		return operations.
			NewGetVariablesClusterKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	return operations.NewGetVariablesClusterKeyOK().WithPayload(&models.Variable{
		Key:   params.Key,
		Value: value,
	})
}

func (ws *webServer) swaggerGetVariablesClusterHandler(params operations.GetVariablesClusterParams) middleware.Responder {
	vars, err := ws.ds.ListClusterVariables()
	if err != nil {
		return operations.
			NewGetVariablesClusterInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	var payload models.Variables
	for k, v := range vars {
		payload = append(payload, &models.Variable{
			Key:   k,
			Value: v,
		})
	}
	return operations.NewGetVariablesClusterOK().WithPayload(payload)
}

func (ws *webServer) swaggerGetVariablesNodesMacHandler(params operations.GetVariablesNodesMacParams) middleware.Responder {
	// GetVariablesNodesMac
	mac, err := net.ParseMAC(params.Mac)
	if err != nil {
		return operations.
			NewGetVariablesNodesMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	vars, err := ws.ds.GetEtcdMachine(mac).ListVariables()
	if err != nil {
		return operations.
			NewGetVariablesNodesMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	var payload models.Variables
	for k, v := range vars {
		payload = append(payload, &models.Variable{
			Key:   k,
			Value: v,
		})
	}
	return operations.NewGetVariablesNodesMacOK().WithPayload(payload)
}

func (ws *webServer) swaggerGetVariablesNodesMacKeyHandler(params operations.GetVariablesNodesMacKeyParams) middleware.Responder {
	// GetVariablesNodesMacKey
	mac, err := net.ParseMAC(params.Mac)
	if err != nil {
		return operations.
			NewGetVariablesNodesMacKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	value, err := ws.ds.GetEtcdMachine(mac).GetVariable(params.Key)
	if client.IsKeyNotFound(err) {
		return operations.
			NewGetVariablesNodesMacKeyNotFound()
	}
	if err != nil {
		return operations.
			NewGetVariablesNodesMacKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	return operations.NewGetVariablesNodesMacKeyOK().WithPayload(&models.Variable{
		Key:   params.Key,
		Value: value,
	})
}

func (ws *webServer) swaggerGetWorkspaceMacHandler(params operations.GetWorkspaceMacParams) middleware.Responder {
	h, err := ws.ds.GetWorkspaceHash()
	if err != nil {
		return operations.
			NewGetWorkspaceMacInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	return operations.NewGetWorkspaceMacOK().WithPayload(&models.Workspace{
		Commit: h,
	})
}

func (ws *webServer) swaggerPostVariablesClusterKeyHandler(params operations.PostVariablesClusterKeyParams) middleware.Responder {
	if err := ws.ds.SetClusterVariable(params.Key, params.Value); err != nil {
		return operations.
			NewPostVariablesClusterKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	return operations.NewPostVariablesClusterKeyOK().WithPayload(&models.Variable{
		Key:   params.Key,
		Value: params.Value,
	})
}

func (ws *webServer) swaggerPostVariablesNodesMacKeyHandler(params operations.PostVariablesNodesMacKeyParams) middleware.Responder {
	mac, err := net.ParseMAC(params.Mac)
	if err != nil {
		return operations.
			NewPostVariablesNodesMacKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	m := ws.ds.GetEtcdMachine(mac)
	if !m.IsExist() {
		return operations.NewPostVariablesNodesMacKeyNotFound()
	}

	if err := m.SetVariable(params.Key, params.Value); err != nil {
		return operations.
			NewPostVariablesNodesMacKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	return operations.NewPostVariablesNodesMacKeyOK().WithPayload(&models.Variable{
		Key:   params.Key,
		Value: params.Value,
	})
}

func (ws *webServer) swaggerDeleteVariablesClusterKeyHandler(params operations.DeleteVariablesClusterKeyParams) middleware.Responder {
	err := ws.ds.DeleteClusterVariable(params.Key)
	if client.IsKeyNotFound(err) {
		return operations.
			NewDeleteVariablesClusterKeyNotFound()
	}

	if err != nil {
		return operations.
			NewDeleteVariablesClusterKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	return operations.NewDeleteVariablesClusterKeyOK()
}

func (ws *webServer) swaggerDeleteVariablesNodesMacKeyHandler(params operations.DeleteVariablesNodesMacKeyParams) middleware.Responder {
	mac, err := net.ParseMAC(params.Mac)
	if err != nil {
		return operations.
			NewDeleteVariablesNodesMacKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	err = ws.ds.GetEtcdMachine(mac).DeleteVariable(params.Key)
	if client.IsKeyNotFound(err) {
		return operations.
			NewDeleteVariablesNodesMacKeyNotFound()
	}
	if err != nil {
		return operations.
			NewDeleteVariablesNodesMacKeyInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	return operations.NewDeleteVariablesNodesMacKeyOK()
}

func (ws *webServer) swaggerPostHeartbeatMacHeartbeatHandler(params operations.PostHeartbeatMacHeartbeatParams) middleware.Responder {
	mac, err := net.ParseMAC(params.Mac)
	if err != nil {
		return operations.
			NewPostHeartbeatMacHeartbeatInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	err = ws.ds.GetEtcdMachine(mac).SetVariable(path.Join("agent_heartbeat"), params.Heartbeat)
	if err != nil {
		return operations.
			NewPostHeartbeatMacHeartbeatInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}

	return operations.NewPostHeartbeatMacHeartbeatOK()
}
