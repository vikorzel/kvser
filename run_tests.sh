#!/bin/bash
curdir=`pwd`
cd ${curdir}/tests/basic
godog basic.feature
cd ${curdir}/tests/qps
godog qps.feature
