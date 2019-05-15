package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	requestsURL   = "https://informacionpublica.paraguay.gov.py/portal-core/rest/solicitudes?limit=20&offset=0&search=%7B%22type%22:%22or%22,%22filters%22:%5B%7B%22path%22:%22descripcion%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22titulo%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22fecha%22,%22sortDesc%22:%22true%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22conteoSuscripciones%22,%22sortDesc%22:%22true%22,%22like%22:%22%22%7D,%7B%22path%22:%22fechaLimite%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22institucion.nombre%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22institucion.ministerio%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22flujosSolicitud.comentario%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22flujosSolicitud.titulo%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22estado.nombre%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22usuario.nombre%22,%22like%22:%22MITIC%22%7D,%7B%22path%22:%22usuario.apellido%22,%22like%22:%22MITIC%22%7D%5D%7D"
	institutionID = 924
)

type PublicInfoTracker struct {
	Reqs []PublicInfoRequest
}

type PublicInfoRequest struct {
	ID            int    `json:"id"`
	Date          string `json:"fecha"`
	RemainingDays int    `json:"diasHabilesFaltantes"`
	Title         string `json:"titulo"`
	State         struct {
		Name string `json:"nombre"`
	} `json:"estado"`
	Institution struct {
		ID int `json:"id"`
	} `json:"institucion"`
	Replied bool
}

func (p *PublicInfoTracker) Init() {
	err := p.FetchUpdates()
	if err != nil {
		log.Print(err)
	}
	for {
		err = p.FetchUpdates()
		if err != nil {
			log.Print(err)
		}
		time.Sleep(12 * time.Hour)
	}
}

func (p *PublicInfoTracker) FetchUpdates() error {
	res, err := http.Get(requestsURL)
	if err != nil {
		return err
	}
	if res.Body == nil {
		return errors.New("Nil body")
	}
	rawBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var reqs []PublicInfoRequest
	err = json.Unmarshal(rawBody, &reqs)
	if err != nil {
		return err
	}
	if len(reqs) == 0 {
		return errors.New("No requests found")
	}
	filteredReqs := []PublicInfoRequest{}
	for _, r := range reqs {
		if r.Institution.ID != institutionID {
			continue
		}
		if r.State.Name == "RESPONDIDO" {
			r.Replied = true
		}
		filteredReqs = append(filteredReqs, r)
	}
	p.Reqs = filteredReqs
	return nil
}
