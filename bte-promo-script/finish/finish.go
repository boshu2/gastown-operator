package main

import (
	"encoding/json"
)

func runFinish(args ...string) error {
	labeled, err := parseFileArgs(args)
	if err != nil {
		return err
	}
	result, err := doUpload(labeled)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return runWebhook(string(raw))
}
