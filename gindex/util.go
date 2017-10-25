package gindex

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"io"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gogits/go-gogs-client"
	"github.com/G-Node/gig"
	"os"
	"path/filepath"
	"strings"
)

func getParsedBody(r *http.Request, obj interface{}) error {
	data, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		log.Debugf("Could not read request body: %+v", err)
		return err
	}
	err = json.Unmarshal(data, obj)
	if err != nil {
		log.Debugf("Could not unmarshal request: %+v, %s", err, string(data))
		return err
	}
	return nil
}

func getParsedResponse(resp *http.Response, obj interface{}) error {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

func getParsedHttpCall(method, path string, body io.Reader, token, csrfT string, obj interface{}) error {
	client := &http.Client{}
	req, _ := http.NewRequest(method, path, body)
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogits=%s; _csrf=%s", token, csrfT))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if (resp.StatusCode != http.StatusOK) {
		return fmt.Errorf("Not Authorized: %d", resp.StatusCode)
	}
	return getParsedResponse(resp, obj)
}

// Encodes a given map into a struct.
// Lazyly Uses json package instead of reflecting directly
func map2struct(in interface{}, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

// Find gin repos under a certain directory, to which the authenticated user has access
func findRepos(rpath string, rbd *ReIndexRequest, gins *GinServer) ([]*gogs.Repository, error) {
	var repos [] *gogs.Repository
	err := filepath.Walk(rpath, func(path string, info os.FileInfo, err error) error {
		if ! info.IsDir() {
			return nil
		}
		repo, err := gig.OpenRepository(path)
		if err != nil {
			return nil
		}
		gRepo, err := hasRepoAccess(repo, rbd, gins)
		if err != nil {
			log.Debugf("no acces to repo:%+v", err)
			return filepath.SkipDir
		}
		repos = append(repos, gRepo)
		return filepath.SkipDir
	})
	return repos, err
}

func hasRepoAccess(repository *gig.Repository, rbd *ReIndexRequest, gins *GinServer) (*gogs.Repository, error) {
	splPath := strings.Split(repository.Path, string(filepath.Separator))
	if ! (len(splPath) > 2) {
		return nil, fmt.Errorf("not a repo path %s", repository.Path)
	}
	owner := splPath[len(splPath)-2]
	name := splPath[len(splPath)-1]
	gRepo := gogs.Repository{}
	err := getParsedHttpCall(http.MethodGet, fmt.Sprintf("%s/api/v1/repos/%s/%s",
		gins.URL, owner, name), nil, rbd.Token, rbd.CsrfT, &gRepo)
	if err != nil {
		return nil, err
	}
	return &gRepo, nil
}