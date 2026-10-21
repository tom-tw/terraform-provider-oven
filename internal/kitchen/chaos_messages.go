package kitchen

// FailureMessages is the catalog of chaos failures. The provider only
// ruins your baking in imagination, never in practice.
var FailureMessages = []string{
	"The gas ran out in the middle of the preparation",
	"A power cut stopped the mixer halfway through",
	"The cat sat on the rising dough",
	"The yeast overslept and forgot to rise",
	"Somebody opened the oven door to peek too early",
	"The flour bag exploded across the whole counter",
	"The double-yolk egg confused all the measurements",
	"The kitchen elf borrowed your whisk and vanished",
	"The smoke alarm demanded a five-minute break",
	"The oven timer batteries died with a final beep",
	"The neighbor borrowed a cup of sugar and never returned it",
	"A sneeze seasoned the batter with black pepper",
	"The butter was too cold to cooperate",
	"The oven light burned out and now nobody can see anything",
	"The recipe card slid behind the counter and is lost",
	"The dough threw a tantrum and needed a time-out in the fridge",
	"The milk turned sour the moment you looked away",
	"The rolling pin rolled off the counter on purpose",
	"The chocolate chips staged a walkout",
	"A mysterious draft ruined the meringue peaks",
}

// MessageFor picks a failure message deterministically from the seed.
func MessageFor(seed string) string {
	return FailureMessages[int(HashRoll("message|"+seed)*float64(len(FailureMessages)))%len(FailureMessages)]
}
