package sparql

import (
	"fmt"
	"log"
	"net/http"
	"bytes"
	"errors"
	"encoding/json"
	"net/url"
)

type Client struct {
	QueryUrl  string
	UpdateUrl string
	http      http.Client
}

type Response struct {
	Results Results `json:"results"`
	Boolean bool    `json:"boolean"`
}

type Results struct {
	Bindings []Binding `json:"bindings"`
}

type Binding map[string]BindingValue

type BindingValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func NewClient(query, update string) *Client {
	return &Client {
		QueryUrl: query,
		UpdateUrl: update,
		http: http.Client{},
	}
}

func (c *Client) AddQuad(context, subject, predicate, object interface{}) error {
	ctx,  err := Literal(context)
	if err != nil { return err }
	
	subj, err := Literal(subject)
	if err != nil { return err }
	
	pred, err := Literal(predicate)
	if err != nil { return err }
	
	obj,  err := Literal(object)
	if err != nil { return err }
	
	q := fmt.Sprintf("INSERT DATA { GRAPH %s { %s %s %s } }", ctx, subj, pred, obj)

	//resp, err := http.Post(c.UpdateUrl, "application/sparql-update", bytes.NewReader([]byte(q)))
	resp, err := http.PostForm(c.UpdateUrl, url.Values{
		"update": []string{q},
	})
	if err != nil {
		log.Printf("UPDATE: %s Failed\n", q)
		return err
	} else {
		log.Printf("UPDATE: %s [%s]\n", q, resp.Status)
		defer resp.Body.Close();
	}
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	} else {
		return errors.New(resp.Status)
	}
}

func (c *Client) Select(query string) (*Response, error) {
	vals := url.Values{
		"query": []string{query},
	}
	
	req, err := http.NewRequest("POST", c.QueryUrl, bytes.NewReader([]byte(vals.Encode())))
	if err != nil {
		return nil, err
	}
	//req.Header.Add("Content-Type", "aplication/sparql-query")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/sparql-results+json")
	
	resp, err := c.http.Do(req)
	
	if err != nil {
		log.Printf("QUERY: %s Failed\n", query)
		return nil, err
	} else {
		log.Printf("QUERY: %s [%s]\n", query, resp.Status)
		defer resp.Body.Close();
	}
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(resp.Status)
	}
	
	var result Response
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

func (c *Client) Update(query string) (*Response, error) {
	vals := url.Values{
		"update": []string{query},
	}
	
	req, err := http.NewRequest("POST", c.UpdateUrl, bytes.NewReader([]byte(vals.Encode())))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")	
	resp, err := c.http.Do(req)
	
	if err != nil {
		log.Printf("UPDATE: %s Failed\n", query)
		return nil, err
	} else {
		log.Printf("UPDATE: %s [%s]\n", query, resp.Status)
		defer resp.Body.Close();
	}
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(resp.Status)
	}
	
	return nil, nil
}