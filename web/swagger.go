package web

import (
	"encoding/json"
	"net"
	"path"
	"strings"

	"github.com/cafebazaar/blacksmith/swagger/models"
	"github.com/cafebazaar/blacksmith/swagger/restapi/operations"
	"github.com/coreos/etcd/client"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

func (ws *webServer) swaggerPostWorkspaceHandler(params operations.PostWorkspaceParams) middleware.Responder {
	// TODO:
	return middleware.NotImplemented("operation PostWorkspace has not yet been implemented")
	// return operations.NewPostWorkspaceInternalServerError()
	// return operations.NewPostWorkspaceOK()
}

func (ws *webServer) swaggerGetNodesHander(params operations.GetNodesParams) middleware.Responder {
	machines, err := ws.ds.MachineInterfaces()
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

		value, err := machine.GetVariable(path.Join("_agent", "heartbeat"))
		if err != nil {
			return operations.
				NewGetNodesInternalServerError().
				WithPayload(&models.Error{err.Error()})
		}
		heatbeat := struct {
			Time   string
			Status string
		}{}
		json.NewDecoder(strings.NewReader(value)).Decode(&heatbeat)
		var t strfmt.Date
		err = t.UnmarshalText([]byte(heatbeat.Time))
		if err != nil {
			return operations.
				NewGetNodesInternalServerError().
				WithPayload(&models.Error{err.Error()})
		}

		resp.Payload = append(resp.Payload, &models.Node{
			IP:            m.IP.String(),
			Mac:           machine.Mac().String(),
			LastHeartbeat: t,
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
	vars, err := ws.ds.MachineInterface(mac).ListVariables()
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
	value, err := ws.ds.MachineInterface(mac).GetVariable(params.Key)
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

func (ws *webServer) swaggerGetWorkspaceHandler(params operations.GetWorkspaceParams) middleware.Responder {
	h, err := ws.ds.GetWorkspaceHash()
	if err != nil {
		return operations.
			NewGetWorkspaceInternalServerError().
			WithPayload(&models.Error{err.Error()})
	}
	return operations.NewGetWorkspaceOK().WithPayload(&models.Workspace{
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

	if err := ws.ds.MachineInterface(mac).SetVariable(params.Key, params.Value); err != nil {
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
	err = ws.ds.MachineInterface(mac).DeleteVariable(params.Key)
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
