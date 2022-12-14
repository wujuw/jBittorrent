package client

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type TrackerClient struct {
	httpClient *http.Client
	trackerUrl string

	info_hash  string
	peer_id    string
	port       int
	uploaded   int
	downloaded int
	left       int
	compact    int
	numwant    int
	event      string
}

func NewTrackerClient(trackerUrl string, info_hash string, peer_id string, port int, uploaded int, downloaded int, left int, compact int, numwant int, event string) *TrackerClient {
	return &TrackerClient{
		httpClient: &http.Client{},
		trackerUrl: trackerUrl,
		info_hash:  info_hash,
		peer_id:    peer_id,
		port:       port,
		uploaded:   uploaded,
		downloaded: downloaded,
		left:       left,
		compact:    compact,
		numwant:    numwant,
		event:      event,
	}
}

func (client *TrackerClient) queryParam() string {
	return fmt.Sprintf("?info_hash=%s&peer_id=%s&port=%d&uploaded=%d&downloaded=%d&left=%d&compact=%d&numwant=%d&event=%s",
		url.QueryEscape(client.info_hash), url.QueryEscape(client.peer_id), client.port, client.uploaded, client.downloaded, client.left, client.compact, client.numwant, client.event)
}

func (client *TrackerClient) queryParamWithoutCompact() string {
	return fmt.Sprintf("?info_hash=%s&peer_id=%s&port=%d&uploaded=%d&downloaded=%d&left=%d&numwant=%d&event=%s",
		url.QueryEscape(client.info_hash), url.QueryEscape(client.peer_id), client.port, client.uploaded, client.downloaded, client.left, client.numwant, client.event)
}

func (client *TrackerClient) Announce() (*TrackerResponse, error) {
	return client.AnnounceWithParams(client.queryParam())
}

// 测试是否能解析无压缩Peers
func (client *TrackerClient) AnnounceWithoutCompact() (*TrackerResponse, error) {
	return client.AnnounceWithParams(client.queryParamWithoutCompact())
}

// 方便测试
func (client *TrackerClient) AnnounceWithParams(urlParams string) (*TrackerResponse, error) {
	res, err := client.httpClient.Get(client.trackerUrl + urlParams)
	if err != nil {
		return nil, err
	} else if res.StatusCode != 200 {
		log.Println(res)
		return nil, errors.New(res.Status)
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
