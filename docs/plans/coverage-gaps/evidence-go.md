# Evidence: Go track

## G01
tdd: TestListSessionsPrunesDead (daemon test hygiene, privateTmp in every port-file/lock/daemon test) — red: with TMPDIR=$X holding rod-cli-devsession.port, `go test ./daemon/ -run TestListSessionsPrunesDead$` deleted it ("RED: dev port file rod-cli-devsession.port deleted by TestListSessionsPrunesDead") — green: same run left it in place ("GREEN: dev port file survived"); `TMPDIR=$X go test ./daemon/ -count=1` passed and left only leakless-* and the planted port file
