package coreapi

import (
	"context"
	"strings"

	core "github.com/ipfs/go-ipfs/core"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	path "github.com/ipfs/go-ipfs/path"

	ipld "gx/ipfs/QmRSU5EqqWVZSNdbU51yXmVoF1uNw3JgTNB6RaiL7DZM16/go-ipld-node"
	cid "gx/ipfs/QmcTcsTvfaeEBRFo1TkFgT8sRmgi1n1LTZpecfVP8fzpGD/go-cid"
)

type CoreAPI struct {
	node *core.IpfsNode
}

func NewCoreAPI(n *core.IpfsNode) coreiface.CoreAPI {
	api := &CoreAPI{n}
	return api
}

func (api *CoreAPI) Unixfs() coreiface.UnixfsAPI {
	return (*UnixfsAPI)(api)
}

func resolve(ctx context.Context, n *core.IpfsNode, ref coreiface.Ref) (ipld.Node, error) {
	switch v := ref.(type) {
	case string:
		if strings.Contains(v, "/") {
			return resolvePath(ctx, n, v)
		}
	}
	return resolveCid(ctx, n, ref)
}

func resolvePath(ctx context.Context, n *core.IpfsNode, p string) (ipld.Node, error) {
	pp, err := path.ParsePath(p)
	if err != nil {
		return nil, err
	}

	node, err := core.Resolve(ctx, n.Namesys, n.Resolver, pp)
	if err != nil {
		return nil, resolveError(err)
	}
	return node, nil
}

func resolveCid(ctx context.Context, n *core.IpfsNode, ref coreiface.Ref) (ipld.Node, error) {
	c, err := cid.Parse(ref)
	if err != nil {
		return nil, err
	}

	node, err := n.DAG.Get(ctx, c)
	if err != nil {
		return nil, resolveError(err)
	}
	return node, nil
}

func resolveError(err error) error {
	if err == core.ErrNoNamesys {
		return coreiface.ErrOffline
	}
	return err
}
