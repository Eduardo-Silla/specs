package sealer

import "math/big"
import "encoding/binary"
import . "github.com/filecoin-project/specs/util"

import filproofs "github.com/filecoin-project/specs/libraries/filcrypto/filproofs"
import file "github.com/filecoin-project/specs/systems/filecoin_files/file"
import sector "github.com/filecoin-project/specs/systems/filecoin_mining/sector"

func (s *SectorSealer_I) SealSector(si SealInputs) *SectorSealer_SealSector_FunRet_I {
	sid := si.SectorID()

	commD := sector.UnsealedSectorCID(s.ComputeDataCommitment(si.UnsealedPath()).As_commD())

	buf := make(Bytes, si.SealCfg().SectorSize())
	f := file.FromPath(si.SealedPath())
	length, _ := f.Read(buf)

	// TODO: How do we meant to handle errors in implementation methods? This could get tedious fast.

	if UInt(length) != UInt(si.SealCfg().SectorSize()) {
		panic("Sector file is wrong size.")
	}

	return &SectorSealer_SealSector_FunRet_I{
		rawValue: Seal(sid, si.RandomSeed(), commD, buf),
	}
}

func (s *SectorSealer_I) VerifySeal(sv sector.SealVerifyInfo) *SectorSealer_VerifySeal_FunRet_I {
	return &SectorSealer_VerifySeal_FunRet_I{}
}

func (s *SectorSealer_I) ComputeDataCommitment(unsealedPath file.Path) *SectorSealer_ComputeDataCommitment_FunRet_I {
	return &SectorSealer_ComputeDataCommitment_FunRet_I{}
}

func ComputeReplicaID(sid sector.SectorID, commD sector.UnsealedSectorCID, seed sector.SealRandomSeed) *SectorSealer_ComputeReplicaID_FunRet_I {

	_, _ = sid.MinerID(), (sid.Number())

	return &SectorSealer_ComputeReplicaID_FunRet_I{}
}

// type SealOutputs struct {
//     SealInfo  sector.SealVerifyInfo
//     ProofAux  sector.ProofAux
// }

// type SealVerifyInfo struct {
//     SectorID
//     OnChain OnChainSealVerifyInfo
// }

// type OnChainSealVerifyInfo struct {
//     UnsealedCID   UnsealedSectorCID  // CommD
//     SealedCID     SealedSectorCID  // CommR
//     RandomSeed    SealRandomSeed
//     Proof         SealProof
//     DealIDs       [deal.DealID]
//     SectorNumber

// }

func SDRParams() *filproofs.StackedDRG_I {
	return &filproofs.StackedDRG_I{}
}

func Seal(sid sector.SectorID, randomSeed sector.SealRandomSeed, commD sector.UnsealedSectorCID, data Bytes) *SealOutputs_I {
	replicaID := ComputeReplicaID(sid, commD, randomSeed).As_replicaID()

	params := SDRParams()

	drg := filproofs.DRG_I{}                // FIXME: Derive from params
	expander := filproofs.ExpanderGraph_I{} // FIXME: Derive from params
	nodeSize := int(params.NodeSize().Size())
	nodes := len(data) / nodeSize
	curveModulus := params.Curve().Modulus()
	layers := int(params.Layers().Layers())
	keyLayers := generateSDRKeyLayers(&drg, &expander, replicaID, nodes, layers, nodeSize, curveModulus)
	key := keyLayers[len(keyLayers)-1]

	replica := encodeData(data, key, nodeSize, curveModulus)

	_ = replica
	return &SealOutputs_I{}
}

func generateSDRKeyLayers(drg *filproofs.DRG_I, expander *filproofs.ExpanderGraph_I, replicaID Bytes, nodes int, layers int, nodeSize int, modulus UInt) []Bytes {
	keyLayers := make([]Bytes, layers)
	var prevLayer Bytes

	for i := 0; i <= layers; i++ {
		keyLayers[i] = labelLayer(drg, expander, replicaID, nodes, nodeSize, prevLayer)
	}
	return keyLayers
}

func labelLayer(drg *filproofs.DRG_I, expander *filproofs.ExpanderGraph_I, replicaID Bytes, nodeSize int, nodes int, prevLayer Bytes) Bytes {
	size := nodes * nodeSize
	labels := make(Bytes, size)

	for i := 0; i < nodes; i++ {
		var dependencies Bytes

		// The first node of every layer has no DRG Parents.
		if i > 0 {
			for parent := range drg.Parents(labels, UInt(i)) {
				start := parent * nodeSize
				dependencies = append(dependencies, labels[start:start+nodeSize]...)
			}
		}

		// The first layer has no expander parents.
		if prevLayer != nil {
			for parent := range expander.Parents(labels, UInt(i)) {
				start := parent * nodeSize
				dependencies = append(dependencies, labels[start:start+nodeSize]...)
			}
		}

		label := generateLabel(replicaID, i, dependencies)
		labels = append(labels, label...)
	}

	return labels
}

func generateLabel(replicaID Bytes, node int, dependencies Bytes) Bytes {
	nodeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nodeBytes, uint64(node))

	preimage := append(replicaID, nodeBytes...)
	preimage = append(preimage, dependencies...)

	return KDF(preimage)
}

// KDF is a key-derivation functions. In SDR, the derived key is used to generate labels directly, without encoding any data.
func KDF(elements Bytes) Bytes {
	return elements // FIXME: Do something.
}

func encodeData(data Bytes, key Bytes, nodeSize int, modulus UInt) Bytes {
	bigMod := big.NewInt(int64(modulus))

	if len(data) != len(key) {
		panic("Key and data must be same length.")
	}

	encoded := make(Bytes, len(data))
	for i := 0; i < len(data); i += nodeSize {
		copy(encoded[i:i+nodeSize], encodeNode(data[i:i+nodeSize], key[i:i+nodeSize], bigMod, nodeSize))
	}

	return encoded
}

func encodeNode(data Bytes, key Bytes, modulus *big.Int, nodeSize int) Bytes {
	// TODO: Allow this to vary by algorithm variant.
	return addEncode(data, key, modulus, nodeSize)
}

func reverse(bytes []byte) {
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
}

func addEncode(data Bytes, key Bytes, modulus *big.Int, nodeSize int) Bytes {
	// FIXME: Check correct endianness.
	sum := new(big.Int)
	reverse(data) // Reverse for little-endian
	reverse(key)  // Reverse for little-endian

	d := new(big.Int).SetBytes(data) // Big-endian
	k := new(big.Int).SetBytes(key)  // Big-endian

	sum = sum.Add(d, k)

	result := new(big.Int)
	resultBytes := result.Mod(sum, modulus).Bytes()[0:nodeSize] // Big-endian
	reverse(resultBytes)                                        // Reverse for little-endian

	return resultBytes
}
