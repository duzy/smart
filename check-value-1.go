//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
//go:build checkpoints

package smart

var checkpoints_value_1 = map[string]map[string]any{
    "loader.go": map[string]any{
        `.test.foo {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word foo}}`:`.test.foo {=compound {3:1:punct .} {3:2:word test} {3:6:punct .} {3:7:word foo}}`,
        `.test.foo {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word foo}}`:`.test.foo {=compound {7:1:punct .} {7:2:word test} {7:6:punct .} {7:7:word foo}}`,
        `.test.1 {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:decimal 1}}`:`.test.1 {=compound {4:1:punct .} {4:2:word test} {4:6:punct .} {4:7:decimal 1}}`,
        `.test.2 {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:decimal 2}}`:`.test.2 {=compound {5:1:punct .} {5:2:word test} {5:6:punct .} {5:7:decimal 2}}`,
        `.test.3 {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:decimal 3}}`:`.test.3 {=compound {8:1:punct .} {8:2:word test} {8:6:punct .} {8:7:decimal 3}}`,
        `.test.4 {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:decimal 4}}`:`.test.4 {=compound {9:1:punct .} {9:2:word test} {9:6:punct .} {9:7:decimal 4}}`,

        `4:9:.test.1 $(.test.foo) {4:20:delegate {3:11:def .test.foo}}`:`- {4:20 {=flag {3:15}}}`,
        `4:9:.test.1 $(.test.foo) {4:62:delegate {3:11:def .test.foo}}`:`- {4:62 {=flag {3:15}}}`,
        `4:9:.test.1 $(.test.foo)foobar {=compound {4:20:delegate {3:11:def .test.foo}} {4:32:word foobar}}`:`-foobar {=compound {4:20 {=flag {3:15}}} {4:32:word foobar}}`,
        `4:9:.test.1 $(.test.foo)foobar {=compound {4:62:delegate {3:11:def .test.foo}} {4:74:word foobar}}`:`-foobar {=compound {4:62 {=flag {3:15}}} {4:74:word foobar}}`,
        `4:9:.test.1 $(equal $(.test.foo)foobar,-foobar) {4:12:delegate {4:14:builtin equal} {=list {=compound {4:20:delegate {3:11:def .test.foo}} {4:32:word foobar}}} {=list {=flag {4:40:word foobar}}}}`:`{=true} {4:12 {4:14:true}}`,
        `4:9:.test.1 -foobar {=compound {4:62 {=flag {3:15}}} {4:74:word foobar}}`:`-foobar {=compound {4:62 {=flag {3:15}}} {4:74:word foobar}}`,
        `4:9:.test.1 $(equal(-str) $(.test.foo)foobar,-foobar) {4:48:delegate {4:50:builtin equal} [{=flag {4:57:word str}}] {=list {=compound {4:62:delegate {3:11:def .test.foo}} {4:74:word foobar}}} {=list {=flag {4:82:word foobar}}}}`:`{=true} {4:48 {4:50:true}}`,

        `5:9:.test.2 $(.test.foo) {5:28:delegate {3:11:def .test.foo}}`:`- {5:28 {=flag {3:15}}}`,
        `5:9:.test.2 $(.test.foo) {5:70:delegate {3:11:def .test.foo}}`:`- {5:70 {=flag {3:15}}}`,
        `5:9:.test.2 $(.test.foo)foobar {=compound {5:28:delegate {3:11:def .test.foo}} {5:40:word foobar}}`:`-foobar {=compound {5:28 {=flag {3:15}}} {5:40:word foobar}}`,
        `5:9:.test.2 $(.test.foo)foobar {=compound {5:70:delegate {3:11:def .test.foo}} {5:82:word foobar}}`:`-foobar {=compound {5:70 {=flag {3:15}}} {5:82:word foobar}}`,
        `5:9:.test.2 $(equal -foobar,$(.test.foo)foobar) {5:12:delegate {5:14:builtin equal} {=list {=flag {5:21:word foobar}}} {=list {=compound {5:28:delegate {3:11:def .test.foo}} {5:40:word foobar}}}}`:`{=true} {5:12 {5:14:true}}`,
        `5:9:.test.2 -foobar {=compound {5:70 {=flag {3:15}}} {5:82:word foobar}}`:`-foobar {=compound {5:70 {=flag {3:15}}} {5:82:word foobar}}`,
        `5:9:.test.2 $(equal(-str) -foobar,$(.test.foo)foobar) {5:48:delegate {5:50:builtin equal} [{=flag {5:57:word str}}] {=list {=flag {5:63:word foobar}}} {=list {=compound {5:70:delegate {3:11:def .test.foo}} {5:82:word foobar}}}}`:`{=true} {5:48 {5:50:true}}`,

        `8:9:.test.3 $(.test.foo) {8:20:delegate {3:11:def .test.foo}}`:`-foo {8:20 {=flag {7:15:word foo}}}`,
        `8:9:.test.3 $(.test.foo) {8:59:delegate {3:11:def .test.foo}}`:`-foo {8:59 {=flag {7:15:word foo}}}`,
        `8:9:.test.3 $(.test.foo)bar {=compound {8:20:delegate {3:11:def .test.foo}} {8:32:word bar}}`:`-foobar {=compound {8:20 {=flag {7:15:word foo}}} {8:32:word bar}}`,
        `8:9:.test.3 $(.test.foo)bar {=compound {8:59:delegate {3:11:def .test.foo}} {8:71:word bar}}`:`-foobar {=compound {8:59 {=flag {7:15:word foo}}} {8:71:word bar}}`,
        `8:9:.test.3 $(equal $(.test.foo)bar,-foobar) {8:12:delegate {8:14:builtin equal} {=list {=compound {8:20:delegate {3:11:def .test.foo}} {8:32:word bar}}} {=list {=flag {8:37:word foobar}}}}`:`{=true} {8:12 {8:14:true}}`,
        `8:9:.test.3 -foobar {=compound {8:59 {=flag {7:15:word foo}}} {8:71:word bar}}`:`-foobar {=compound {8:59 {=flag {7:15:word foo}}} {8:71:word bar}}`,
        `8:9:.test.3 $(equal(-str) $(.test.foo)bar,-foobar) {8:45:delegate {8:47:builtin equal} [{=flag {8:54:word str}}] {=list {=compound {8:59:delegate {3:11:def .test.foo}} {8:71:word bar}}} {=list {=flag {8:76:word foobar}}}}`:`{=true} {8:45 {8:47:true}}`,

        `9:9:.test.4 $(.test.foo) {9:28:delegate {3:11:def .test.foo}}`:`-foo {9:28 {=flag {7:15:word foo}}}`,
        `9:9:.test.4 $(.test.foo) {9:67:delegate {3:11:def .test.foo}}`:`-foo {9:67 {=flag {7:15:word foo}}}`,
        `9:9:.test.4 $(.test.foo)bar {=compound {9:28:delegate {3:11:def .test.foo}} {9:40:word bar}}`:`-foobar {=compound {9:28 {=flag {7:15:word foo}}} {9:40:word bar}}`,
        `9:9:.test.4 $(.test.foo)bar {=compound {9:67:delegate {3:11:def .test.foo}} {9:79:word bar}}`:`-foobar {=compound {9:67 {=flag {7:15:word foo}}} {9:79:word bar}}`,
        `9:9:.test.4 $(equal -foobar,$(.test.foo)bar) {9:12:delegate {9:14:builtin equal} {=list {=flag {9:21:word foobar}}} {=list {=compound {9:28:delegate {3:11:def .test.foo}} {9:40:word bar}}}}`:`{=true} {9:12 {9:14:true}}`,
        `9:9:.test.4 -foobar {=compound {9:67 {=flag {7:15:word foo}}} {9:79:word bar}}`:`-foobar {=compound {9:67 {=flag {7:15:word foo}}} {9:79:word bar}}`,
        `9:9:.test.4 $(equal(-str) -foobar,$(.test.foo)bar) {9:45:delegate {9:47:builtin equal} [{=flag {9:54:word str}}] {=list {=flag {9:60:word foobar}}} {=list {=compound {9:67:delegate {3:11:def .test.foo}} {9:79:word bar}}}}`:`{=true} {9:45 {9:47:true}}`,
    },
}

var checkstrs_value_1 = map[string]map[string]any{
}
