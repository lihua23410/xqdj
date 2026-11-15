package ospf

import (
	"sort"
	"sync"
)

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

// lsdb 是一张链路状态表。锁和记录放在同一块上：浅拷贝会连锁一起带走，
// 深拷贝则整张换新。沙盘拆开网络那把锁之后，两边仍不会并发读写同一张表。
type lsdb struct {
	mu  sync.Mutex
	rec database
}

func (r *router) table() *lsdb {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.db == nil {
		r.db = &lsdb{}
	}
	return r.db
}

func (r *router) learn(a announcement) bool {
	db := r.table()
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.rec[a.lsaKey].sequence >= a.sequence {
		return false
	}
	if db.rec == nil {
		db.rec = make(database)
	}
	db.rec[a.lsaKey] = a
	return true
}

func (r *router) sequence(k lsaKey) uint64 {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	db := r.db
	r.mu.Unlock()
	if db == nil {
		return 0
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.rec[k].sequence
}

func (r *router) record(k lsaKey) announcement {
	if r == nil {
		return announcement{}
	}
	r.mu.Lock()
	db := r.db
	r.mu.Unlock()
	if db == nil {
		return announcement{}
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.rec[k]
}

func (r *router) recordCount() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	db := r.db
	r.mu.Unlock()
	if db == nil {
		return 0
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return len(db.rec)
}

func (r *router) records() []announcement {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	db := r.db
	r.mu.Unlock()
	if db == nil {
		return nil
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.rec.ordered()
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
			db := r.table()
			db.mu.Lock()
			for key := range db.rec {
				if key.advertiser == abr && isSummary(key.kind) {
					delete(db.rec, key)
				}
			}
			db.mu.Unlock()
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
