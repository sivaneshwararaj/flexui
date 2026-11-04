# M7 review images

These are **expected** images, rendered headlessly on Linux. They are NOT
screenshots of the macOS window, which has not been run (see
`platform/darwin/README.md`).

* `stage2_expected.png`: what stage 2 must show at 480x320 points, scale
  1 (four quadrants: red, green / blue, white; black border).
* `stage3_expected.png`, `stage3_expected@2x.png`: what stage 3 must
  show; stage 3 compares its presented frame with this render byte for
  byte and prints `identical: true|false`.
