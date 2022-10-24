package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type TrackerClient struct {
	httpClient *http.Client
	trackerUrl string

	info_hash string
	peer_id string
	port int
	uploaded int
	downloaded int
	left int
	compact int
	numwant int
	event string
}

func NewTrackerClient(trackerUrl string, info_hash string, peer_id string, port int, uploaded int, downloaded int, left int, compact int, numwant int, event string) *TrackerClient {
	return &TrackerClient{
		httpClient: &http.Client{},
		trackerUrl: trackerUrl,
		info_hash: info_hash,
		peer_id: peer_id,
		port: port,
		uploaded: uploaded,
		downloaded: downloaded,
		left: left,
		compact: compact,
		numwant: numwant,
		event: event,
	}
}

func (client *TrackerClient) queryParam() (string) {
	return fmt.Sprintf("?info_hash=%s&peer_id=%s&port=%d&uploaded=%d&downloaded=%d&left=%d&compact=%d&numwant=%d&event=%s",
		url.QueryEscape(client.info_hash), url.QueryEscape(client.peer_id), client.port, client.uploaded, client.downloaded, client.left, client.compact, client.numwant, client.event)
}

func (client *TrackerClient) Announce() (*TrackerResponse, error) {
	res, err := client.httpClient.Get(client.trackerUrl + client.queryParam())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	trackerResponse, err := ParseTrackerResponse(body)
	if err != nil {
		return nil, err
	}
	return trackerResponse, nil
}

