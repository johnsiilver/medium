package service

import (
	"context"
	"sort"
	"testing"

	pb "github.com/johnsiilver/medium/grpc_testing/proto"
	"github.com/kylelemons/godebug/pretty"
)

func TestServers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		req     *pb.ServersReq
		want    []string
		wantErr bool
	}{
		{
			desc:    "Regex won't compile",
			req:     &pb.ServersReq{NameFilterRe: `(what`},
			wantErr: true,
		},
		{
			desc: "Use regex to get subset of servers",
			req:  &pb.ServersReq{NameFilterRe: `\d+server`},
			want: []string{"1server", "2server"},
		},
		{
			desc: "Use datacenter filter to get servers from a single datacenter",
			req:  &pb.ServersReq{DatacenterFilter: []string{"ab"}},
			want: []string{"1server", "2server", "server40"},
		},
		{
			desc: "Use datacenter filter a name regex",
			req:  &pb.ServersReq{NameFilterRe: `server\d+`, DatacenterFilter: []string{"ab"}},
			want: []string{"server40"},
		},
		{
			desc: "Get everything",
			req:  &pb.ServersReq{},
			want: []string{
				"server1",
				"server2",
				"server3",
				"server4",
				"server5",
				"server6",
				"server7",
				"server8",
				"server9",
				"server10",
				"1server",
				"2server",
				"server40",
			},
		},
	}

	// Create our service with our test data.
	auth := New("../data/test/data.json")

	for _, test := range tests {
		ch := auth.servers(context.Background(), test.req)

		var err error
		var got []string
		for msg := range ch {
			if msg.err != nil {
				err = msg.err
				continue
			}
			got = append(got, msg.data.Name)
		}

		// This simply checks if we got an error when we expected one.
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestServers(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestServers(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil: // Since we got an expected error, we can skip the rest of the test.
			continue
		}

		// Sort the results so we can compare them.
		sort.Strings(test.want)
		sort.Strings(got)

		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("TestServers(%s): -want/+got:\n%s", test.desc, diff)
		}

	}
}
