package sv

var (
	qvMarginCalcJS = `
	// XXX: PLACEHOLDER, DO NOT USE
	function calcMargin(currentTally, voteUser, voteChoice, targetVote) {

		var currentVote = 0.0;
		var currentCharge = 0.0;
	
		var currentScoresByUser = currentTally.scores_by_user[voteUser];
		if (currentScoresByUser !== undefined) {
			var currentChoiceByUser = currentScoresByUser[voteChoice];
			if (currentChoiceByUser !== undefined) {
				currentVote = currentChoiceByUser.score;
				currentCharge = currentChoiceByUser.strength;
			}
		}
	
		var targetCharge = targetVote * targetVote;
		var cost = targetCharge - currentCharge;
	
		return {
			"help": {
				"label": "Help",
				"description": "This ballot uses a standard Quadratic Voting.",
				"value": null,
			},
			"currentVote": {
				"label": "Current vote",
				"description": "Your current vote",
				"value": currentVote,
			},
			"targetVote": {
				"label": "Target vote",
				"description": "Your target vote",
				"value": targetVote,
			},
			"cost": {
				"label": "Cost",
				"description": "Cost of changing your vote",
				"value": cost,
			},
		}
	}
	`
)
