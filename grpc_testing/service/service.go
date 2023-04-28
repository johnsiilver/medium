package service

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"regexp"

	pb "github.com/johnsiilver/medium/grpc_testing/proto"
)

// Authority is our gRPC service implementation.
type Authority struct {
	dataFilePath string

	pb.UnimplementedAuthorityServer // required by gRPC
}

// New creates a new Authority using the data in dataFilePath.
func New(dataFilePath string) *Authority {
	return &Authority{dataFilePath: dataFilePath}
}

type streamMsg[T any] struct {
	data T
	err  error
}

// Servers implements the gRPC AuthorityServer.Servers() method.
func (a *Authority) Servers(req *pb.ServersReq, stream pb.Authority_ServersServer) error {
	for msg := range a.servers(stream.Context(), req) {
		if msg.err != nil {
			return msg.err
		}
		if err := stream.Send(&pb.ServerMsg{Name: msg.data.Name}); err != nil {
			return err
		}
	}
	return nil
}

type serversRec struct {
	Name       string
	Datacenter string
}

// servers returns a channel that will stream the servers that match the request.
func (a *Authority) servers(ctx context.Context, req *pb.ServersReq) <-chan streamMsg[*pb.ServerMsg] {
	ch := make(chan streamMsg[*pb.ServerMsg], 1)

	filter, err := newFilter(req)
	if err != nil {
		ch <- streamMsg[*pb.ServerMsg]{err: err}
		close(ch)
		return ch
	}

	f, err := os.Open(a.dataFilePath)
	if err != nil {
		ch <- streamMsg[*pb.ServerMsg]{err: err}
		close(ch)
		return ch
	}

	// Stream our entries if they match the various filters in the request.
	go func() {
		defer f.Close()
		defer close(ch)
		a.streamServer(ctx, json.NewDecoder(f), ch, filter)
	}()

	return ch
}

// streamServer streams the servers from the file inside dec, filters them, and sends matches to ch.
func (a *Authority) streamServer(ctx context.Context, dec *json.Decoder, ch chan<- streamMsg[*pb.ServerMsg], filter filter) {
	for {
		var r serversRec
		if err := dec.Decode(&r); err == io.EOF {
			return
		} else if err != nil {
			ch <- streamMsg[*pb.ServerMsg]{err: err}
			return
		}

		if !filter.match(r) {
			continue
		}

		select {
		case <-ctx.Done():
			ch <- streamMsg[*pb.ServerMsg]{err: ctx.Err()}
			return
		case ch <- streamMsg[*pb.ServerMsg]{data: &pb.ServerMsg{Name: r.Name}}:
		}
	}
}

// filter is used to filter a serversRec.
type filter struct {
	nameFilter *regexp.Regexp
	dcs        map[string]bool
}

// match returns true if the serversRec matches the filter.
func (f filter) match(rec serversRec) bool {
	// If the name filter was defined, reject any entries that don't match.
	if f.nameFilter != nil && !f.nameFilter.MatchString(rec.Name) {
		return false
	}

	// If the DC filter was provided, reject any entries that don't match.
	if f.dcs != nil && !f.dcs[rec.Datacenter] {
		return false
	}
	return true
}

// newFilter creates a new filter object that can be used to test a serversRec matches the filters
// specified in the request.
func newFilter(req *pb.ServersReq) (filter, error) {
	f := filter{}

	// Compile our name filter if it was given.
	if req.NameFilterRe != "" {
		var err error
		f.nameFilter, err = regexp.Compile(req.NameFilterRe)
		if err != nil {
			return filter{}, err
		}
	}

	// dcs if not nil has keys that are the datacenters we want to filter for.
	if len(req.DatacenterFilter) > 0 {
		f.dcs = map[string]bool{}
		for _, dc := range req.DatacenterFilter {
			f.dcs[dc] = true
		}
	}
	return f, nil
}
