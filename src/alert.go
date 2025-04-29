package miner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func (miner *Miner) Alert(data ...any) {
	fmtted := []string{}
	for _, d := range data {
		fmtted = append(fmtted, spew.Sdump(d))
		spew.Dump(d)
	}

	if miner.Options.DebugWebhook == "" {
		return
	}

	result := strings.Join(fmtted, "\n")
	for {
		// send to discord webhook
		// if the result is over 2000 bytes, send as file attachment
		var err error
		var req *http.Request
		if len(result) < 2000 {
			body := map[string]any{
				"content": result,
			}
			var encoded []byte
			encoded, err = json.Marshal(body)
			if err != nil {
				return
			}
			req, err = http.NewRequest("POST", miner.Options.DebugWebhook, bytes.NewBuffer(encoded))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
			}
		} else {
			// send as file attachment using multipart/form-data
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			var part io.Writer
			part, err = writer.CreateFormFile("file", "alert.txt")
			if err != nil {
				return
			}
			_, err = part.Write([]byte(result))
			if err != nil {
				return
			}
			err = writer.Close()
			if err != nil {
				return
			}
			req, err = http.NewRequest("POST", miner.Options.DebugWebhook, body)
			if err == nil {
				req.Header.Set("Content-Type", writer.FormDataContentType())
			}
		}

		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}

		res, err := miner.DefaultUser.GraphQL.Client.Do(req)
		if err != nil {
			fmt.Println("Error sending alert:", err)
			return
		}

		if res.StatusCode >= http.StatusBadRequest {
			fmt.Println("Error sending alert, status code:", res.StatusCode)
			body, _ := io.ReadAll(res.Body)
			fmt.Println("Response body:", string(body))
			res.Body.Close()
			time.Sleep(time.Second)
			continue
		}
		res.Body.Close()
		break
	}
}
