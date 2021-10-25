package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/pat"
)

func validateName(name string) bool {
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

func validateTag(tag string) bool {
	for _, r := range tag {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			if r < '0' || r > '9' {
				if r != '.' {
					return false
				}
			}
		}
	}
	return true
}

func validateDigest(digest string) bool {
	for _, r := range digest {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			if r != ':' {
				if r < '0' || r > '9' {
					return false
				}
			}
		}
	}
	return true
}

func PrepRepo(repo string) error {
	err := os.MkdirAll(repo+"/tags", 0744)
	if err != nil {
		return err
	}
	err = os.MkdirAll(repo+"/blobs", 0744)
	if err != nil {
		return err
	}
	err = os.MkdirAll(repo+"/uploads", 0744)
	if err != nil {
		return err
	}
	return nil
}

func Accepts(r *http.Request, t string) bool {
	for _, hi := range r.Header.Values("Accept") {
		for _, val := range strings.Split(hi, ",") {
			if strings.TrimSpace(val) == t {
				return true
			}
		}

	}
	return false
}

func main() {

	p := pat.New()
	p.Post("/v2/{repo}/manifests/{tag}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	p.Get("/v2/{repo}/manifests/{tag}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get(":repo")
		tag := r.URL.Query().Get(":tag")

		if !(validateName(repo) && (validateTag(tag) || validateDigest(tag))) {
			// Really dont have time for things
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if _, err := os.Stat(repo + "/tags/" + tag); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if Accepts(r, "application/vnd.oci.image.manifest.v1+json") {
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		} else if Accepts(r, "application/vnd.docker.distribution.manifest.v2+json") {
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		} else {
			for _, ac := range r.Header.Values("Accept") {
				log.Printf("Client accepts %s", ac)
			}
			log.Panicf("Unable to find acceptable response type")
		}

		fileBytes, err := ioutil.ReadFile(repo + "/tags/" + tag)
		if err != nil {
			panic(err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(fileBytes)
	})
	p.Get("/v2/{repo}/blobs/{digest}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get(":repo")
		digest := r.URL.Query().Get(":digest")

		if !(validateName(repo) && validateDigest(digest)) {
			// Really dont have time for things
			return
		}

		f, err := os.Open(repo + "/blobs/" + digest)
		if err != nil {
			log.Panic(err)
		}
		io.Copy(w, f)
		w.WriteHeader(http.StatusOK)
	})
	p.Put("/v2/{repo}/blobs/uploads/{tag}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get(":repo")
		tag := r.URL.Query().Get(":tag")
		digest := r.URL.Query().Get("digest")

		f, err := os.Open(repo + "/uploads/" + tag)
		if err != nil {
			log.Panic(err)
		}

		hash := sha256.New()
		if _, err := io.Copy(hash, f); err != nil {
			log.Panic(err)
		}
		sum := hash.Sum(nil)

		if digest == fmt.Sprintf("sha256:%x", sum) {
			log.Printf("We got a good blob \\o/")
			os.Rename(repo+"/uploads/"+tag, repo+"/blobs/"+digest)
		}

		w.WriteHeader(http.StatusCreated)

	})
	p.Put("/v2/{repo}/manifests/{tag}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get(":repo")
		tag := r.URL.Query().Get(":tag")

		if !(validateName(repo) && validateTag(tag)) {
			log.Print("Repo / Tag invalid")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err := PrepRepo(repo)
		if err != nil {
			log.Panic(err)
		}

		w.WriteHeader(http.StatusOK)

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Panic(err)
		}

		f, err := os.Create(repo + "/tags/" + tag)
		if err != nil {
			log.Panic(err)
		}
		defer f.Close()

		_, err = f.Write(data)
		if err != nil {
			log.Panic(err)
		}

		hash := sha256.New()
		hash.Write(data)
		sum := hash.Sum(nil)

		if _, err := os.Stat(repo + "/tags/" + fmt.Sprintf("sha256:%x", sum)); os.IsNotExist(err) {
			err = os.Symlink(tag, repo+"/tags/"+fmt.Sprintf("sha256:%x", sum))
			if err != nil {
				log.Panic(err)
			}
		}

	})
	p.Head("/v2/{repo}/manifests/{tag}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get(":repo")
		tag := r.URL.Query().Get(":tag")
		if !(validateName(repo) && (validateTag(tag) || validateDigest(tag))) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if _, err := os.Stat(repo + "/tags/" + tag); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		}
		w.WriteHeader(http.StatusOK)
	})
	p.Head("/v2/{repo}/blobs/{digest}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get(":repo")
		digest := r.URL.Query().Get(":digest")

		if !(validateName(repo) && validateDigest(digest)) {
			return
		}

		if _, err := os.Stat(repo + "/blobs/" + digest); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
			// path/to/whatever does not exist
		}

		w.WriteHeader(http.StatusOK)
	})
	p.Get("/v2/", func(w http.ResponseWriter, r *http.Request) {
		return
	})
	p.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World\n")
	})

	p.Patch("/v2/{repo}/blobs/uploads/{guid}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get(":repo")
		guid := r.URL.Query().Get(":guid")

		if !(validateName(repo)) {
			return
		}

		err := PrepRepo(repo)
		if err != nil {
			log.Panic(err)
		}

		tmpfile, err := os.Create(repo + "/uploads/" + guid)
		if err != nil {
			log.Panic(err)
		}
		defer tmpfile.Close()
		_, err = io.Copy(tmpfile, r.Body)
		if err != nil {
			log.Panic(err)
		}

		w.Header().Add("Location", "/v2/"+repo+"/blobs/uploads/"+guid)
		w.WriteHeader(http.StatusAccepted)
	})
	p.Post("/v2/{repo}/blobs/uploads/", func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get(":repo")

		if !(validateName(repo)) {
			return
		}

		w.Header().Add("Location", "/v2/"+repo+"/blobs/uploads/"+uuid.New().String())
		w.WriteHeader(http.StatusAccepted)
	})

	log.Fatal(http.ListenAndServe(":5000", handlers.LoggingHandler(os.Stdout, p)))
}
