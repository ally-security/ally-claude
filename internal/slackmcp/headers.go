package slackmcp

import (
	"encoding/json"
	"fmt"
	"os"
)

func PrintHeadersJSON() error {
	token, err := LoadUserToken()
	if err != nil {
		return err
	}
	out, err := json.Marshal(map[string]string{
		"Authorization": "Bearer " + token,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(out))
	return err
}
