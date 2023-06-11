package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

var (
	gitserver = server.DefaultServer
	gitpath   = "/tmp/usertest/test.git"
)

func runHTTP(dir, addr string) error {
	http.HandleFunc("/info/refs", httpInfoRefs)
	http.HandleFunc("/git-upload-pack", httpGitUploadPack)
	http.HandleFunc("/git-receive-pack", httpGitReceivePack)

	log.Println("starting http server on", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func httpInfoRefs(rw http.ResponseWriter, r *http.Request) {
	logger(rw, r)
	service := r.URL.Query().Get("service")
	rw.Header().Set("Content-Type", fmt.Sprintf("application/x-%v-advertisement", service))

	var err error
	var session transport.Session

	ep, err := transport.NewEndpoint(gitpath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if service == "git-upload-pack" {
		session, err = gitserver.NewUploadPackSession(ep, nil)
	} else if service == "git-receive-pack" {
		session, err = gitserver.NewReceivePackSession(ep, nil)
	} else {
		http.Error(rw, "service type not recognized", http.StatusBadRequest)
	}

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	ar, err := session.AdvertisedReferencesContext(r.Context())
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	ar.Prefix = [][]byte{
		[]byte(fmt.Sprintf("# service=%v", service)),
		pktline.Flush,
	}
	err = ar.Encode(rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func httpGitUploadPack(rw http.ResponseWriter, r *http.Request) {
	logger(rw, r)
	rw.Header().Set("Content-Type", "application/x-git-upload-pack-result")

	upr := packp.NewUploadPackRequest()
	err := upr.Decode(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	ep, err := transport.NewEndpoint(gitpath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	sess, err := gitserver.NewUploadPackSession(ep, nil)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	res, err := sess.UploadPack(r.Context(), upr)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	err = res.Encode(rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func httpGitReceivePack(w http.ResponseWriter, r *http.Request) {
	logger(w, r)
	w.Header().Set("Content-Type", "application/x-git-receive-pack-result")

	rer := packp.NewReferenceUpdateRequest()
	err := rer.Decode(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	ep, err := transport.NewEndpoint(gitpath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	sess, err := gitserver.NewReceivePackSession(ep, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	res, err := sess.ReceivePack(r.Context(), rer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	err = res.Encode(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func logger(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)
}
