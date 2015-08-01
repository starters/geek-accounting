// +build appengine

package deb

import (
	"fmt"
	"time"

	"appengine"
	"appengine/datastore"
)

type datastoreSpace struct{}

type blockWrapper struct {
	Block dataBlock
	AsOf  time.Time
}

type keyWrapper struct {
	key  *datastore.Key
	asOf time.Time
}

func NewDatastoreSpace(ctx appengine.Context, key *datastore.Key) (Space, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ctx is nil")
	}
	if key == nil {
		key = datastore.NewIncompleteKey(ctx, "space", nil)
		var err error
		if key, err = datastore.Put(ctx, key, &datastoreSpace{}); err != nil {
			return nil, err
		}
	}
	var ls *largeSpace
	errc := make(chan error, 1)
	in := func() chan *dataBlock {
		c := make(chan *dataBlock)
		go func() {
			var err error
			defer close(c)
			defer func() { errc <- err }()
			q := datastore.NewQuery("data_block").Ancestor(key)
			t := q.Run(ctx)
			for {
				bw := blockWrapper{*ls.newDataBlock(), time.Now()}
				var k *datastore.Key
				k, err = t.Next(&bw)
				if err == datastore.Done {
					err = nil
					break
				}
				if err != nil {
					break
				}
				bw.Block.key = keyWrapper{k, bw.AsOf}
				c <- &bw.Block
			}
		}()
		return c
	}
	out := make(chan *dataBlock)
	go func() {
		for block := range out {
			bw := blockWrapper{*block, time.Now()}
			if block.key == nil || block.key.(keyWrapper).key == nil {
				block.key = keyWrapper{datastore.NewIncompleteKey(ctx, "data_block", key),
					time.Now()}
			} else {
				var bw2 blockWrapper
				if err := datastore.Get(ctx, block.key.(keyWrapper).key, &bw2); err != nil {
					errc <- err
					continue
				}
				if bw2.AsOf != block.key.(keyWrapper).asOf {
					errc <- fmt.Errorf("Concurrent modification")
					continue
				}
			}
			_, err := datastore.Put(ctx, block.key.(keyWrapper).key, &bw)
			errc <- err
		}
	}()
	ls = newLargeSpace(1014*1024, in, out, errc)
	return ls, nil
}
