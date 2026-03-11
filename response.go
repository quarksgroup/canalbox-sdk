package canalbox

import (
	"encoding/json"
	"fmt"
)

type APIError struct {
	Message string `json:"message"`
}

type Action struct {
	Id                string          `json:"id"`
	Descriptor        string          `json:"descriptor"`
	CallingDescriptor string          `json:"callingDescriptor"`
	ReturnValue       json.RawMessage `json:"returnValue"`
	Error             []APIError      `json:"error"`
	State             string          `json:"state"`
}

type Response struct {
	Actions []Action `json:"actions"`
}

func firstAction(resp *Response) (*Action, error) {
	if resp == nil || len(resp.Actions) == 0 {
		return nil, fmt.Errorf("no actions in aura response")
	}

	action := resp.Actions[0]
	if action.State != "SUCCESS" {
		if len(action.Error) > 0 {
			if isSessionExpiredMessage(action.Error[0].Message) {
				return nil, wrapSessionExpired(action.Error[0].Message)
			}

			return nil, fmt.Errorf("aura action error: %s", action.Error[0].Message)
		}
		return nil, fmt.Errorf("aura action in state %q", action.State)
	}

	return &action, nil
}

func decodeReturnValue(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty return value")
	}

	var wrapper struct {
		ReturnValue json.RawMessage `json:"returnValue"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.ReturnValue) > 0 {
		return decodeReturnValue(wrapper.ReturnValue, target)
	}

	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return decodeReturnValue(json.RawMessage(str), target)
	}

	return json.Unmarshal(raw, target)
}
