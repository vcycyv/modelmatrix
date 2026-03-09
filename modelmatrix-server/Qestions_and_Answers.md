## In the Model Performance tab, clicking Set Baseline shows default values for metrics like accuracy. Are these values computed, and if so, how? 

How are they calculated?
1. Where they come from

When you open “Set Baseline Metrics”, the dialog is filled from model.metrics (see PerformanceMonitorPanel.tsx around 71–80 and DetailPanel.tsx around 775: modelMetrics={model.metrics}).
model.metrics is set when the model is created from a completed build, using the metrics sent in the build callback from the compute service (see createModelFromBuild in modelmatrix-server/internal/module/build/application/service.go around 524–525: Metrics: callback.Metrics).
So the numbers in the form are exactly the metrics that were computed at training time and then stored on the model.

2. Where they are actually computed

They are computed in modelmatrix-compute during the training job:

After training, the trainer evaluates on the test set:
model_trainer.py (around 121–122):
metrics = algo.evaluate(model, X_test, y_test, model_type)
The algorithm’s evaluate() (e.g. in modelmatrix-compute/src/algorithms/base.py around 64–67) uses sklearn on the test predictions and labels:
Accuracy: accuracy_score(y_test, y_pred) → (TP + TN) / total
Precision: precision_score(y_test, y_pred, average="weighted", zero_division=0) → TP / (TP + FP) per class, then weighted
Recall: recall_score(y_test, y_pred, average="weighted", zero_division=0) → TP / (TP + FN) per class, then weighted
F1: f1_score(y_test, y_pred, average="weighted", zero_division=0) → harmonic mean of precision and recall, weighted
So the formulas are the standard ML ones (confusion-matrix based); the implementation is sklearn’s with average="weighted" for multi-class.

3. End-to-end flow

Build runs → compute service trains the model and evaluates on the train/test split.
Compute service sends a callback to the server with a metrics map (accuracy, precision, recall, f1_score, etc.).
Server stores those on the build, then copies them onto the model when it creates the model from the build.
UI loads the model and passes model.metrics into the Performance Monitor as modelMetrics.
The Set Baseline dialog initializes its fields from modelMetrics when the form is empty (so “Your model training metrics are pre-filled below”).
So: the numbers are calculated as test-set metrics during training (sklearn, weighted precision/recall/F1), then stored on the model and used to pre-fill the baseline form. You can change them in the form before saving if you want a different baseline.