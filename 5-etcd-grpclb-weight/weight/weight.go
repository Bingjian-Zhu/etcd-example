package weight

import (
	"math/rand"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
)

// Name is the name of weight balancer.
const Name = "weight"

// DefaultNodeWeight is 1
var DefaultNodeWeight = 1

// attributeKey is the type used as the key to store AddrInfo in the Attributes
// field of resolver.Address.
type attributeKey struct{}

// AddrInfo will be stored inside Address metadata in order to use weighted balancer.
type AddrInfo struct {
	Weight uint32
}

// SetAddrInfo returns a copy of addr in which the Attributes field is updated
// with addrInfo.
//
// This is an EXPERIMENTAL API.
func SetAddrInfo(addr resolver.Address, addrInfo AddrInfo) resolver.Address {
	addr.Attributes = addr.Attributes.WithValues(attributeKey{}, addrInfo)
	return addr
}

// GetAddrInfo returns the AddrInfo stored in the Attributes fields of addr.
//
// This is an EXPERIMENTAL API.
func GetAddrInfo(addr resolver.Address) AddrInfo {
	v := addr.Attributes.Value(attributeKey{})
	ai, _ := v.(AddrInfo)
	return ai
}

// NewBuilder creates a new weight balancer builder.
func NewBuilder() balancer.Builder {
	return base.NewBalancerBuilderV2(Name, &rrPickerBuilder{}, base.Config{HealthCheck: true})
}

type rrPickerBuilder struct{}

func (*rrPickerBuilder) Build(info base.PickerBuildInfo) balancer.V2Picker {
	grpclog.Infof("weightPicker: newPicker called with info: %v", info)
	if len(info.ReadySCs) == 0 {
		return base.NewErrPickerV2(balancer.ErrNoSubConnAvailable)
	}
	var scs []balancer.SubConn
	for sc := range info.ReadySCs {
		scs = append(scs, sc)
	}
	return &rrPicker{
		subConns: scs,
	}
}

type rrPicker struct {
	// subConns is the snapshot of the roundrobin balancer when this picker was
	// created. The slice is immutable. Each Get() will do a round robin
	// selection from it and return the selected SubConn.
	subConns []balancer.SubConn

	mu sync.Mutex
}

func (p *rrPicker) Pick(balancer.PickInfo) (balancer.PickResult, error) {
	var totalWeight int
	p.mu.Lock()
	for _, addr := range p.subConns {
		val := GetAddrInfo(addr)
		if val.Weight == 0 {
			val.Weight = DefaultNodeWeight
		}
		totalWeight += val.Weight
	}

	curWeight := rand.Intn(totalWeight)
	curIndex := -1
	for index, addr := range p.subConns {
		node := GetAddrInfo(addr)
		curWeight -= node.Weight
		if curWeight < 0 {
			curIndex = index
			break
		}
	}

	if curIndex == -1 {
		return
	}
	sc := p.subConns[curIndex]
	p.mu.Unlock()
	return balancer.PickResult{SubConn: sc}, nil
}
