package ospf

import "sort"

type lsa uint8

const (
	routerLSA   lsa = 1
	networkLSA  lsa = 2
	summaryLSA  lsa = 3
	asbrLSA     lsa = 4
	externalLSA lsa = 5
	nssaLSA     lsa = 7
)

// Areas represent internal prefixes. A node identifies a router destination.
// The fighter injects versions of one external prefix into its own network.
type destination struct {
	area      int
	node      *router
	isDefault bool
}

type lsaKey struct {
	kind        lsa
	destination destination
	advertiser  *router
}

type announcement struct {
	lsaKey
	sequence   uint64
	propagate  bool
	forwarding *router
}

type database map[lsaKey]announcement

func (r *router) learn(a announcement) bool {
	if r.db[a.lsaKey].sequence >= a.sequence {
		return false
	}
	if r.db == nil {
		r.db = make(database)
	}
	r.db[a.lsaKey] = a
	return true
}

func (db database) ordered() []announcement {
	records := make([]announcement, 0, len(db))
	for _, a := range db {
		records = append(records, a)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].sequence < records[j].sequence })
	return records
}

func (n *network) originate(kind lsa, advertiser *router, dest destination) announcement {
	n.sequence++
	return announcement{
		lsaKey:   lsaKey{kind: kind, destination: dest, advertiser: advertiser},
		sequence: n.sequence,
	}
}

func (n *network) inject(from point, asbr *router, out *commands) {
	kind := externalLSA
	if asbr.area == 3 {
		kind = nssaLSA
	}
	a := n.originate(kind, asbr, destination{})
	if kind == nssaLSA {
		a.propagate, a.forwarding = true, asbr
	}
	n.transmit(from, asbr, round(a, asbr.area), out)
}

// Losing the simulated ABR attachment withdraws its summaries immediately.
// Keep a watermark so an already-flying copy cannot restore them.
func (n *network) withdrawSummaries(abr *router) {
	abr.summarySince = n.sequence + 1
	clear(abr.imported)
	for _, area := range n.areas {
		for _, r := range area {
			for key := range r.db {
				if key.advertiser == abr && isSummary(key.kind) {
					delete(r.db, key)
				}
			}
			for key := range r.imported {
				if key.advertiser == abr {
					delete(r.imported, key)
				}
			}
		}
	}
}

func (n *network) activeSummary(a announcement) bool {
	abr := a.advertiser
	return n.bridges[abr.area].abr == abr && a.sequence >= abr.summarySince
}
