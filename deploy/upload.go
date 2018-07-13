package deploy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
)

const uploadUrl = "https://uploads.github.com/repos/%s/%s/releases/%v/assets?name=%s"

func UploadAsset(owner, repo string, releaseId int64, file, token string) error {
	b, e := ioutil.ReadFile(file)
	if e != nil {
		return e
	}

	_, name := filepath.Split(file)
	url := fmt.Sprintf(uploadUrl, owner, repo, releaseId, name)

	rq, e := http.NewRequest("POST", fmt.Sprintf(url, name), bytes.NewBuffer(b))
	if e != nil {
		return e
	}
	defer rq.Body.Close()

	rq.Header.Set("Authorization", "token "+token)
	rq.Header.Set("Content-Type", "application/octet-stream")

	rs, e := http.DefaultClient.Do(rq)
	if e != nil {
		return e
	}
	defer rs.Body.Close()

	if rs.StatusCode != 200 {
		buf := &bytes.Buffer{}
		io.Copy(buf, rs.Body)
		log.Println(buf.String())
		return fmt.Errorf(rs.Status)
	}

	return nil
}
