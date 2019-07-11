package main

// the core algorithm here was borrowed from:
// Blake Mizerany's `noeqd` https://github.com/bmizerany/noeqd
// and indirectly:
// Twitter's `snowflake` https://github.com/twitter/snowflake

// only minor cleanup and changes to introduce a type, combine the concept 
// of workerId + datacenterId into a single identifier, and modify the 
// behavior when sequences rollover for our specific implementation needs

import (
	"encoding/hex"
	"errors"
	"time"
)

const (
	workerIdBits   = uint64(10)
	sequenceBits   = uint64(12)
	workerIdShift  = sequenceBits
	timestampShift = sequenceBits + workerIdBits
	sequenceMask   = int64(-1) ^ (int64(-1) << sequenceBits)

	// Tue, 21 Mar 2006 20:50:14.000 GMT
	twepoch = int64(1288834974657)
)

var sequence int64
var lastTimestamp int64

type GUID int64

func NewGUID(workerId int64) (GUID, error) {
	ts := time.Now().UnixNano() / 1e6

	if ts < lastTimestamp {
		return 0, errors.New("E_TIME_BACKWARDS")
	}

	if lastTimestamp == ts {
		sequence = (sequence + 1) & sequenceMask
		if sequence == 0 {
			return 0, errors.New("E_SEQUENCE_EXPIRED")
		}
	} else {
		sequence = 0
	}

	lastTimestamp = ts

	id := ((ts - twepoch) << timestampShift) |
		(workerId << workerIdShift) |
		sequence

	return GUID(id), nil
}

func (g GUID) Hex() []byte {
	b := make([]byte, 8)
	b[0] = byte(g >> 56)
	b[1] = byte(g >> 48)
	b[2] = byte(g >> 40)
	b[3] = byte(g >> 32)
	b[4] = byte(g >> 24)
	b[5] = byte(g >> 16)
	b[6] = byte(g >> 8)
	b[7] = byte(g)

	h := make([]byte, 16)
	hex.Encode(h, b)

	return h
}
