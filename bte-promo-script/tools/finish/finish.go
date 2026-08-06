package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func runFinish(args ...string) error {
	env, err := loadJobEnv()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "promo-finish finish: job=%s request_id=%s api=%s\n",
		env.JobName, env.RequestID, env.APIURL)

	labeled, err := parseFileArgs(args)
	if err != nil {
		return err
	}
	for _, lf := range labeled {
		fmt.Fprintf(os.Stderr, "promo-finish finish: artifact key=%q path=%q\n", lf.Key, lf.Path)
	}

	fmt.Fprintf(os.Stderr, "promo-finish finish: step 1/2 upload → promo-api\n")
	result, err := doUpload(labeled)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "promo-finish finish: step 2/2 webhook → promo-api → GenAxis PROMO_SCRIPT_COMPLETED\n")
	if err := runWebhook(string(raw)); err != nil {
		return err
	}

	copyDevOutput(env.JobName, labeled)
	return nil
}
