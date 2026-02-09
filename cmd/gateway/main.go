package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	gateway "github.com/kasuboski/luakit/pkg/gateway"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/frontend/gateway/grpcclient"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/moby/buildkit/util/bklog"
	_ "github.com/moby/buildkit/util/grpcutil/encoding/proto"
)

const (
	Package  = "luakit"
	Version  = "0.1.0"
	Revision = "dev"
)

func main() {
	var version bool
	flag.BoolVar(&version, "version", false, "show version")
	flag.Parse()

	if version {
		fmt.Printf("%s %s %s %s\n", os.Args[0], Package, Version, Revision)
		os.Exit(0)
	}

	if err := grpcclient.RunFromEnvironment(appcontext.Context(), func(ctx context.Context, c gwclient.Client) (*gwclient.Result, error) {
		return gateway.Build(ctx, c)
	}); err != nil {
		bklog.L.Errorf("fatal error: %+v", err)
		panic(err)
	}
}
