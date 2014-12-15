package epictest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	blockservice "github.com/jbenet/go-ipfs/blockservice"
	bitswap "github.com/jbenet/go-ipfs/exchange/bitswap"
	tn "github.com/jbenet/go-ipfs/exchange/bitswap/testnet"
	importer "github.com/jbenet/go-ipfs/importer"
	chunk "github.com/jbenet/go-ipfs/importer/chunk"
	merkledag "github.com/jbenet/go-ipfs/merkledag"
	path "github.com/jbenet/go-ipfs/path"
	mockrouting "github.com/jbenet/go-ipfs/routing/mock"
	uio "github.com/jbenet/go-ipfs/unixfs/io"
	util "github.com/jbenet/go-ipfs/util"
	errors "github.com/jbenet/go-ipfs/util/debugerror"
	delay "github.com/jbenet/go-ipfs/util/delay"
	"github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-random"
)

const kSeed = 1

func Test100MBInstantaneous(t *testing.T) {
	t.Log("a sanity check")

	t.Parallel()

	conf := Config{
		NetworkLatency:    0,
		RoutingLatency:    0,
		BlockstoreLatency: 0,
		DataAmountBytes:   100 * 1024 * 1024,
	}

	AddCatBytes(conf)
}

func TestDegenerateSlowBlockstore(t *testing.T) {
	SkipUnlessEpic(t)
	t.Parallel()

	conf := Config{BlockstoreLatency: 50 * time.Millisecond}

	if err := AddCatPowers(conf, 128); err != nil {
		t.Fatal(err)
	}
}

func TestDegenerateSlowNetwork(t *testing.T) {
	SkipUnlessEpic(t)
	t.Parallel()

	conf := Config{NetworkLatency: 400 * time.Millisecond}

	if err := AddCatPowers(conf, 128); err != nil {
		t.Fatal(err)
	}
}

func TestDegenerateSlowRouting(t *testing.T) {
	SkipUnlessEpic(t)
	t.Parallel()

	conf := Config{RoutingLatency: 400 * time.Millisecond}

	if err := AddCatPowers(conf, 128); err != nil {
		t.Fatal(err)
	}
}

func Test100MBMacbookCoastToCoast(t *testing.T) {
	SkipUnlessEpic(t)
	t.Parallel()

	conf := Config{
		DataAmountBytes: 100 * 1024 * 1024,
	}.Network_NYtoSF().Blockstore_SlowSSD2014().Routing_Slow()

	if err := AddCatBytes(conf); err != nil {
		t.Fatal(err)
	}
}

func AddCatPowers(conf Config, megabytesMax int64) error {
	var i int64
	for i = 1; i < megabytesMax; i = i * 2 {
		fmt.Printf("%d MB\n", i)
		conf.DataAmountBytes = i * 1024 * 1024
		if err := AddCatBytes(conf); err != nil {
			return err
		}
	}
	return nil
}

func AddCatBytes(conf Config) error {

	sessionGenerator := bitswap.NewSessionGenerator(
		tn.VirtualNetwork(delay.Fixed(conf.NetworkLatency)), // TODO rename VirtualNetwork
		mockrouting.NewServerWithDelay(delay.Fixed(conf.RoutingLatency)),
	)
	defer sessionGenerator.Close()

	adder := sessionGenerator.Next()
	catter := sessionGenerator.Next()
	catter.SetBlockstoreLatency(conf.BlockstoreLatency)

	adder.SetBlockstoreLatency(0) // disable blockstore latency during add operation
	var data bytes.Buffer
	random.WritePseudoRandomBytes(conf.DataAmountBytes, &data, kSeed) // FIXME get a lazy reader
	keyAdded, err := add(adder, bytes.NewReader(data.Bytes()))
	if err != nil {
		return err
	}
	adder.SetBlockstoreLatency(conf.BlockstoreLatency) // add some blockstore delay to make the catter wait

	readerCatted, err := cat(catter, keyAdded)
	if err != nil {
		return err
	}

	// verify
	var bufout bytes.Buffer
	io.Copy(&bufout, readerCatted)
	if 0 != bytes.Compare(bufout.Bytes(), data.Bytes()) {
		return errors.New("catted data does not match added data")
	}
	return nil
}

func cat(catter bitswap.Instance, k util.Key) (io.Reader, error) {
	catterdag := merkledag.NewDAGService(&blockservice.BlockService{catter.Blockstore(), catter.Exchange})
	nodeCatted, err := (&path.Resolver{catterdag}).ResolvePath(k.String())
	if err != nil {
		return nil, err
	}
	return uio.NewDagReader(nodeCatted, catterdag)
}

func add(adder bitswap.Instance, r io.Reader) (util.Key, error) {
	nodeAdded, err := importer.BuildDagFromReader(
		r,
		merkledag.NewDAGService(&blockservice.BlockService{adder.Blockstore(), adder.Exchange}),
		nil,
		chunk.DefaultSplitter,
	)
	if err != nil {
		return "", err
	}
	return nodeAdded.Key()
}

func SkipUnlessEpic(t *testing.T) {
	if os.Getenv("IPFS_EPIC_TEST") == "" {
		t.SkipNow()
	}
}