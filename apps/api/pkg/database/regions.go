package database

type continent string

const (
	continentUs   continent = "us"
	continentEu   continent = "eu"
	continentAsia continent = "asia"
)

func getClosestContinent(flyRegion string) continent {
	switch flyRegion {
	case "atl":
	case "bog":
	case "bos":
	case "den":
	case "dfw":
	case "ewr":
	case "iad":
	case "lax":
	case "mia":
	case "ord":
	case "qro":
	case "scl":
	case "sea":
	case "sjc":
	case "yul":
	case "yyz":
		return continentUs
	case "ams":
	case "arn":
	case "cdg":
	case "eze":
	case "fra":
	case "gdl":
	case "gig":
	case "gru":
	case "jnb":
	case "lhr":
	case "mad":
	case "otp":
	case "waw":
		return continentEu
	case "hkg":
	case "bom":
	case "nrt":
	case "sin":
	case "syd":
		return continentAsia
	}
	return continentUs
}
