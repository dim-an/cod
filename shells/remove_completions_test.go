// Copyright 2020 Dmitry Ermolov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shells

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestBashRemoveCompletions(t *testing.T) {
	completeOut := `complete -F _longopt touch
complete -o bashdefault -o filenames -F __cod_complete_bash qu
complete -F _longopt ldd
complete -F _minimal ./qu
complete -F _command then
complete -W 'foo bar' qu
complete -F _command command
complete -F _longopt sha384sum
complete -F _known_hosts fping6
`
	completions, err := BashRemoveCompletions("qu", strings.NewReader(completeOut))
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"complete -r ./qu",
			"complete -r qu",
			"complete -W 'foo bar' qu",
		},
		completions,
	)
}
