package dag

import (
	"sync"

	"github.com/opencontainers/go-digest"
	"google.golang.org/protobuf/proto"
)

const (
	initialCacheCapacity = 1024
	maxCacheSize         = 4096
)

var (
	digestCache    = make(map[string]digest.Digest, initialCacheCapacity)
	digestMu       sync.RWMutex
	marshalCache   = make(map[string][]byte, initialCacheCapacity)
	marshalCacheMu sync.RWMutex
)

var (
	deterministicOpts = proto.MarshalOptions{Deterministic: true}
)

func (n *OpNode) Digest() digest.Digest {
	if n == nil || n.op == nil {
		return ""
	}

	if n.digest != "" {
		return digest.Digest(n.digest)
	}

	dt, err := deterministicOpts.Marshal(n.op)
	if err != nil {
		return ""
	}

	digestMu.Lock()
	if d, ok := digestCache[string(dt)]; ok {
		digestMu.Unlock()
		n.digest = string(d)
		return d
	}
	d := digest.FromBytes(dt)
	if len(digestCache) < maxCacheSize {
		digestCache[string(dt)] = d
	}
	digestMu.Unlock()

	n.digest = string(d)
	return d
}

func (n *OpNode) DigestString() string {
	if n == nil || n.op == nil {
		return ""
	}

	if n.digest != "" {
		return n.digest
	}

	d := n.Digest()
	n.digest = string(d)
	return n.digest
}

func (n *OpNode) InvalidateDigest() {
	if n != nil {
		n.digest = ""
	}
}

func (n *OpNode) MarshalOp() ([]byte, error) {
	if n == nil || n.op == nil {
		return nil, nil
	}

	key := n.DigestString()

	marshalCacheMu.RLock()
	if dt, ok := marshalCache[key]; ok {
		marshalCacheMu.RUnlock()
		return dt, nil
	}
	marshalCacheMu.RUnlock()

	dt, err := deterministicOpts.Marshal(n.op)
	if err != nil {
		return nil, err
	}

	marshalCacheMu.Lock()
	if len(marshalCache) < maxCacheSize {
		marshalCache[key] = dt
	}
	marshalCacheMu.Unlock()

	return dt, nil
}

func ClearDigestCache() {
	digestMu.Lock()
	digestCache = make(map[string]digest.Digest, initialCacheCapacity)
	digestMu.Unlock()

	marshalCacheMu.Lock()
	marshalCache = make(map[string][]byte, initialCacheCapacity)
	marshalCacheMu.Unlock()
}
