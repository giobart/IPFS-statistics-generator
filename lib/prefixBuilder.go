package lib

import (
	"github.com/ipfs/go-cid"
	"math"
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
