# M5 review images

Produced by `TestCounterClicks` (`app_test.go`) with synthetic events in
the headless platform (`headless.Session`):

* `counter_0.png`: first frame (count 0).
* `counter_3.png`: after three clicks on "+" (count 3; the note appears
  because the template shows it for count >= 3).
* `counter_3@2x.png`: the same at device scale 2.

After one more click on "−" the incrementally updated frame is checked to
be byte-identical to a fresh render of the same state.
