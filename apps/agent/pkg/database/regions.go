package database

type continent string

const (
	continentUs   continent = "us"
	continentEu   continent = "eu"
	continentAsia continent = "asia"
)

func getClosestContinent(flyRegion string) continent {
	switch flyRegion {
	case "atl", "bog", "bos", "den", "dfw", "ewr", "iad", "lax", "mia", "ord", "qro", "scl", "sea", "sjc", "yul", "yyz":
		return continentUs
	case "ams", "arn", "cdg", "eze", "fra", "gdl", "gig", "gru", "jnb", "lhr", "mad", "otp", "waw":
		return continentEu
	case "hkg", "bom", "nrt", "sin", "syd":
		return continentAsia
	}
	return continentUs
}
