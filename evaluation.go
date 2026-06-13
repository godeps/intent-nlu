package intentnlu

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"

	"github.com/jbrukh/bayesian"
)

type tokenizedSample struct {
	Text   string
	Intent string
	Tokens []string
}

type splitResult struct {
	Train []tokenizedSample
	Val   []tokenizedSample
	Test  []tokenizedSample
}

func preprocessSamples(samples []Sample, tok *Tokenizer, aliases map[string]string, unknownIntent string) ([]tokenizedSample, error) {
	if tok == nil {
		return nil, fmt.Errorf("tokenizer is nil")
	}
	processed := make([]tokenizedSample, 0, len(samples))
	for _, s := range samples {
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		intent := strings.TrimSpace(s.Intent)
		if intent == "" {
			continue
		}
		intent = NormalizeIntent(intent, aliases)
		if intent == "" || intent == unknownIntent {
			continue
		}
		tokens := tok.Tokenize(text)
		if len(tokens) == 0 {
			continue
		}
		processed = append(processed, tokenizedSample{Text: text, Intent: intent, Tokens: tokens})
	}
	if len(processed) == 0 {
		return nil, fmt.Errorf("no valid tokenized samples")
	}
	return processed, nil
}

func splitTokenizedSamples(samples []tokenizedSample, cfg DatasetSplitConfig) splitResult {
	if len(samples) == 0 {
		return splitResult{}
	}
	if !cfg.Enabled {
		return splitResult{Train: append([]tokenizedSample(nil), samples...)}
	}
	trainRatio, valRatio, testRatio := normalizeSplitRatios(cfg.TrainRatio, cfg.ValRatio, cfg.TestRatio)
	seed := cfg.Seed
	if seed == 0 {
		seed = 42
	}

	type keyedSample struct {
		hash   uint64
		sample tokenizedSample
	}
	keyed := make([]keyedSample, 0, len(samples))
	for _, sample := range samples {
		hasher := fnv.New64a()
		_, _ = hasher.Write([]byte(sample.Text))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(sample.Intent))
		_, _ = hasher.Write([]byte(fmt.Sprintf("|%d", seed)))
		keyed = append(keyed, keyedSample{hash: hasher.Sum64(), sample: sample})
	}
	sort.SliceStable(keyed, func(i, j int) bool {
		if keyed[i].hash == keyed[j].hash {
			if keyed[i].sample.Intent == keyed[j].sample.Intent {
				return keyed[i].sample.Text < keyed[j].sample.Text
			}
			return keyed[i].sample.Intent < keyed[j].sample.Intent
		}
		return keyed[i].hash < keyed[j].hash
	})

	sortedSamples := make([]tokenizedSample, 0, len(samples))
	for _, item := range keyed {
		sortedSamples = append(sortedSamples, item.sample)
	}

	n := len(sortedSamples)
	trainN := int(math.Round(float64(n) * trainRatio))
	valN := int(math.Round(float64(n) * valRatio))
	if trainN < 1 {
		trainN = 1
	}
	if trainN > n {
		trainN = n
	}
	if trainN+valN > n {
		valN = n - trainN
	}
	testN := n - trainN - valN
	if testN < 0 {
		testN = 0
	}

	if valRatio > 0 && valN == 0 && n-trainN > 0 {
		valN = 1
		if trainN+valN > n {
			valN = n - trainN
		}
		testN = n - trainN - valN
		if testN < 0 {
			testN = 0
		}
	}
	if testRatio > 0 && testN == 0 && n-trainN-valN > 0 {
		testN = 1
		if trainN+valN+testN > n {
			if valN > 0 {
				valN--
			} else if trainN > 1 {
				trainN--
			}
		}
	}

	result := splitResult{}
	result.Train = append(result.Train, sortedSamples[:trainN]...)
	start := trainN
	if valN > 0 {
		result.Val = append(result.Val, sortedSamples[start:start+valN]...)
		start += valN
	}
	if start < n {
		result.Test = append(result.Test, sortedSamples[start:]...)
	}
	return result
}

func normalizeSplitRatios(train, val, test float64) (float64, float64, float64) {
	if train <= 0 {
		train = 0.8
	}
	if val < 0 {
		val = 0
	}
	if test < 0 {
		test = 0
	}
	total := train + val + test
	if total <= 0 {
		return 1, 0, 0
	}
	return train / total, val / total, test / total
}

func trainClassifier(samples []tokenizedSample) (*bayesian.Classifier, []string, error) {
	if len(samples) == 0 {
		return nil, nil, fmt.Errorf("training samples are empty")
	}
	intentSet := make(map[string]struct{})
	for _, sample := range samples {
		intentSet[sample.Intent] = struct{}{}
	}
	if len(intentSet) < 2 {
		return nil, nil, fmt.Errorf("bayesian classifier requires at least two intents")
	}
	classes := make([]string, 0, len(intentSet))
	for intent := range intentSet {
		classes = append(classes, intent)
	}
	sort.Strings(classes)

	bayesClasses := make([]bayesian.Class, 0, len(classes))
	for _, class := range classes {
		bayesClasses = append(bayesClasses, bayesian.Class(class))
	}
	classifier := bayesian.NewClassifier(bayesClasses...)
	for _, sample := range samples {
		classifier.Learn(sample.Tokens, bayesian.Class(sample.Intent))
	}
	return classifier, classes, nil
}

func calibrateThresholds(
	classifier *bayesian.Classifier,
	valSamples []tokenizedSample,
	classes []string,
	defaultThreshold float64,
) map[string]float64 {
	thresholds := make(map[string]float64, len(classes))
	if len(classes) == 0 {
		return thresholds
	}
	if len(valSamples) == 0 {
		for _, intent := range classes {
			thresholds[intent] = defaultThreshold
		}
		return thresholds
	}

	type sampleScore struct {
		Actual    string
		Predicted string
		Score     float64
	}

	scores := make([]sampleScore, 0, len(valSamples))
	for _, sample := range valSamples {
		bestClass, probs, _ := classifier.ClassifyProb(sample.Tokens)
		bestIntent := string(bestClass)
		bestScore := 0.0
		for i, cls := range classifier.Classes {
			if string(cls) == bestIntent {
				bestScore = probs[i]
				break
			}
		}
		scores = append(scores, sampleScore{
			Actual:    sample.Intent,
			Predicted: bestIntent,
			Score:     bestScore,
		})
	}

	for _, intent := range classes {
		candidates := []float64{defaultThreshold, 0.01, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9}
		positiveCount := 0
		for _, sample := range scores {
			if sample.Actual == intent {
				positiveCount++
			}
			if sample.Predicted == intent {
				candidates = append(candidates, sample.Score)
			}
		}
		if positiveCount == 0 {
			thresholds[intent] = defaultThreshold
			continue
		}

		unique := uniqueSortedThresholds(candidates)
		bestThreshold := defaultThreshold
		bestF1 := -1.0
		bestPrecision := -1.0
		bestRecall := -1.0
		for _, threshold := range unique {
			tp, fp, fn := 0, 0, 0
			for _, sample := range scores {
				accepted := sample.Predicted == intent && sample.Score >= threshold
				if accepted {
					if sample.Actual == intent {
						tp++
					} else {
						fp++
					}
				} else if sample.Actual == intent {
					fn++
				}
			}
			precision, recall, f1 := precisionRecallF1(tp, fp, fn)
			if f1 > bestF1+1e-12 ||
				(math.Abs(f1-bestF1) <= 1e-12 && precision > bestPrecision+1e-12) ||
				(math.Abs(f1-bestF1) <= 1e-12 && math.Abs(precision-bestPrecision) <= 1e-12 && recall > bestRecall+1e-12) ||
				(math.Abs(f1-bestF1) <= 1e-12 && math.Abs(precision-bestPrecision) <= 1e-12 && math.Abs(recall-bestRecall) <= 1e-12 && threshold < bestThreshold) {
				bestThreshold = threshold
				bestF1 = f1
				bestPrecision = precision
				bestRecall = recall
			}
		}
		thresholds[intent] = bestThreshold
	}
	return thresholds
}

func uniqueSortedThresholds(values []float64) []float64 {
	set := map[float64]struct{}{}
	for _, v := range values {
		if v <= 0 {
			continue
		}
		if v > 1 {
			v = 1
		}
		set[v] = struct{}{}
	}
	if len(set) == 0 {
		return []float64{0.5}
	}
	result := make([]float64, 0, len(set))
	for v := range set {
		result = append(result, v)
	}
	sort.Float64s(result)
	return result
}

func evaluateClassifier(
	classifier *bayesian.Classifier,
	samples []tokenizedSample,
	classes []string,
	unknownIntent string,
	defaultThreshold float64,
	thresholds map[string]float64,
	split string,
) EvalReport {
	report := EvalReport{
		Split:     split,
		Samples:   len(samples),
		PerIntent: map[string]ClassMetrics{},
		Confusion: map[string]map[string]int{},
	}
	if len(samples) == 0 {
		return report
	}

	labelSet := make(map[string]struct{}, len(classes)+1)
	for _, intent := range classes {
		labelSet[intent] = struct{}{}
	}
	labelSet[unknownIntent] = struct{}{}

	correct := 0
	unknownCount := 0
	candidateHits := make(map[string]candidateStats, len(classes))
	for _, sample := range samples {
		bestClass, probs, _ := classifier.ClassifyProb(sample.Tokens)
		updateCandidateStats(candidateHits, classifier, sample.Intent, probs)
		predictedIntent := string(bestClass)
		bestScore := 0.0
		for i, cls := range classifier.Classes {
			if string(cls) == predictedIntent {
				bestScore = probs[i]
				break
			}
		}

		threshold := defaultThreshold
		if value, ok := thresholds[predictedIntent]; ok && value > 0 {
			threshold = value
		}
		if bestScore < threshold {
			predictedIntent = unknownIntent
			unknownCount++
		}

		actualIntent := sample.Intent
		if report.Confusion[actualIntent] == nil {
			report.Confusion[actualIntent] = map[string]int{}
		}
		report.Confusion[actualIntent][predictedIntent]++
		labelSet[actualIntent] = struct{}{}
		labelSet[predictedIntent] = struct{}{}

		if predictedIntent == actualIntent {
			correct++
		}
	}

	report.Accuracy = float64(correct) / float64(len(samples))
	report.UnknownRate = float64(unknownCount) / float64(len(samples))

	intents := make([]string, 0, len(classes))
	for _, intent := range classes {
		if intent == unknownIntent {
			continue
		}
		intents = append(intents, intent)
	}
	sort.Strings(intents)

	allStats := computeAllConfusionStats(report.Confusion)
	totalTP := 0
	totalFP := 0
	totalFN := 0
	macroSum := 0.0
	macroCount := 0
	for _, intent := range intents {
		st := allStats[intent]
		precision, recall, f1 := precisionRecallF1(st.tp, st.fp, st.fn)
		report.PerIntent[intent] = ClassMetrics{
			Precision:       precision,
			Recall:          recall,
			F1:              f1,
			Top1Recall:      candidateHits[intent].recallAt(1),
			Top3Recall:      candidateHits[intent].recallAt(3),
			Top5Recall:      candidateHits[intent].recallAt(5),
			Support:         st.support,
			TP:              st.tp,
			FP:              st.fp,
			FN:              st.fn,
			Top1CandidateTP: candidateHits[intent].top1,
			Top3CandidateTP: candidateHits[intent].top3,
			Top5CandidateTP: candidateHits[intent].top5,
		}
		totalTP += st.tp
		totalFP += st.fp
		totalFN += st.fn
		macroSum += f1
		macroCount++
	}

	if macroCount > 0 {
		report.MacroF1 = macroSum / float64(macroCount)
	}
	_, _, microF1 := precisionRecallF1(totalTP, totalFP, totalFN)
	report.MicroF1 = microF1
	return report
}

type intentStats struct {
	tp, fp, fn, support int
}

type candidateStats struct {
	top1, top3, top5, support int
}

func (s candidateStats) recallAt(k int) float64 {
	if s.support == 0 {
		return 0
	}
	switch k {
	case 1:
		return float64(s.top1) / float64(s.support)
	case 3:
		return float64(s.top3) / float64(s.support)
	case 5:
		return float64(s.top5) / float64(s.support)
	default:
		return 0
	}
}

func computeAllConfusionStats(confusion map[string]map[string]int) map[string]intentStats {
	result := make(map[string]intentStats, len(confusion))
	for actual, row := range confusion {
		for predicted, count := range row {
			// actual row: support and TP/FN
			st := result[actual]
			st.support += count
			if predicted == actual {
				st.tp += count
			} else {
				st.fn += count
			}
			result[actual] = st

			// predicted column: FP (only when actual != predicted)
			if predicted != actual {
				pst := result[predicted]
				pst.fp += count
				result[predicted] = pst
			}
		}
	}
	return result
}

func updateCandidateStats(stats map[string]candidateStats, classifier *bayesian.Classifier, actualIntent string, probs []float64) {
	if classifier == nil || len(probs) != len(classifier.Classes) {
		return
	}
	actualScore := 0.0
	found := false
	for i, cls := range classifier.Classes {
		if string(cls) == actualIntent {
			actualScore = probs[i]
			found = true
			break
		}
	}
	if !found {
		return
	}

	rank := 1
	for i, score := range probs {
		intent := string(classifier.Classes[i])
		if intent == actualIntent {
			continue
		}
		if score > actualScore || (score == actualScore && intent < actualIntent) {
			rank++
		}
	}

	st := stats[actualIntent]
	st.support++
	if rank <= 1 {
		st.top1++
	}
	if rank <= 3 {
		st.top3++
	}
	if rank <= 5 {
		st.top5++
	}
	stats[actualIntent] = st
}

func precisionRecallF1(tp, fp, fn int) (precision, recall, f1 float64) {
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return precision, recall, f1
}
