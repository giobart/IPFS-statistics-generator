package lib

import (
	"github.com/ipfs/go-cid"
	u "github.com/ipfs/go-ipfs-util"
	ks "github.com/ipsn/go-ipfs/gxlibs/github.com/whyrusleeping/go-keyspace"
	sha256 "github.com/minio/sha256-simd"
	"math"
	"math/bits"
	"math/rand"
	"strconv"
	"time"
)

/* given a peerId and a bucket number "i" this function generate a cid that belongs
to the i-th bucket for the given peer*/
func BucketPrefixBuilder(peerid string, distance int) (string, error) {

	//persing of the cid
	id, err := cid.Decode(peerid)
	if err != nil {
		return "", err
	}

	//conversion to byte
	byteId := id.Bytes()

	//taking last 32 bytes
	byteHead := byteId[:len(byteId)-32]
	byteTrail := byteId[len(byteId)-32:]

	//choosing the byte to modify
	byteNum := int(math.Floor(float64(distance) / 8))

	//generating a bitmask in order to modify the bit i-th bit of the choosen byte
	bitMask := int(math.Pow(2, float64(distance-(byteNum*8))))

	//apply the bitask to the byte in order to generate the new id
	byteTrail[31-byteNum] = byte((int(byteTrail[31-byteNum]) + bitMask) % 256)

	//merging cid prefix with peer id
	byteId = append(byteHead, byteTrail...)

	//generating a new cid from the bytes
	_, newCid, err := cid.CidFromBytes(byteId)
	if err != nil {
		return "", err
	}

	return newCid.String(), nil
}

func GenerateBucketQuery(iterations int, startingCid string) ([256]string, error) {

	var result [256]string

	rand.Seed(int64(time.Now().Nanosecond()))

	//persing of the cid
	id, err := cid.Decode(startingCid)
	if err != nil {
		return result, err
	}

	//original cid hash
	originalHash := sha256.Sum256(id.Bytes())

	//conversion to byte
	byteId := id.Bytes()

	//taking last 32 bytes
	byteHead := byteId[:len(byteId)-32]
	byteTrail := byteId[len(byteId)-32:]

	for i := 0; i < iterations; i++ {
		//create random byte sequence on byte trail
		for j := 0; j < len(byteTrail); j++ {
			randb := rand.Int() % 256
			byteTrail[j] = byte(randb)
		}

		//create new cid from the random sequence
		byteId = append(byteHead, byteTrail...)
		_, newCid, err := cid.CidFromBytes(byteId)
		if err != nil {
			log.Error(err)
		}

		//hash the generated cid
		newHash := sha256.Sum256(newCid.Bytes())

		//measure XOR bucket distance
		dist := ks.ZeroPrefixLen(u.XOR(originalHash[:], newHash[:]))

		//if this is a new cid, set it to the bucket
		if result[dist] == "" {
			result[dist] = newCid.String()
		}
		if dist > 20 {
			println(dist)
		}

	}

	return result, nil

}

// returning distance between 2 cid string
func cidCompare(cidstring1 string, cidstring2 string) int {
	//parsing cid1
	cid1, err := cid.Decode(cidstring1)
	if err != nil {
		log.Error(err)
	}

	//parsing cid2
	cid2, err := cid.Decode(cidstring2)
	if err != nil {
		log.Error(err)
	}

	//generating hash
	hash1 := sha256.Sum256(cid1.Bytes())
	hash2 := sha256.Sum256(cid2.Bytes())

	//calculating distance
	dist := ks.ZeroPrefixLen(u.XOR(hash1[:], hash2[:]))

	log.Info("Distance " + strconv.Itoa(dist))
	return dist
}

// number of consecutive zeros in a byte array
func prefixLen(id []byte) int {
	for i, b := range id {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}
	return len(id) * 8
}
