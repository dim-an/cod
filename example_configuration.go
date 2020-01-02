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

package main

var ExampleConfiguration = `# cod configuration
# Put this configuration into '~/.config/cod/config.toml'.
# 
# Lines starting with '#' are comments.

#
# Rules
# =====

# Configuration might have several '[[rule]]' sections.
# Whenever cod detects usage of help command in the shell it will scan all
# such sections. When it finds first appropriate rule (see 'executable' key)
# 'policy' from this rule is used. If no appropriate rule is found default
# policy is used.

# 'executable' controls which executables this rule is applied to.
# 'executble' might have one of following forms:
#   - '/path/to/executable' :: current rule applies to specified executable
#   - '/path/to/dir/*' :: current rule applies to all executables
#                         in '/path/to/dir' directory but not in
#                         its subdirectories.
#   - '/path/to/dir/**' :: current rule applies to all executables
#                          in '/path/to/dir' directory or its subdirectory
#   - 'exec-name' :: current rule applies to all executables with basename
#                    'exec-name'
#
# Paths must be absolute. '~/' is expanded to home directory.

# 'policy' controls cod actions whenever it detects usage of help command
# in the shell. Possible values for 'policy' are:
#   - 'ask'    :: default policy, cod will ask if you want to learn command;
#   - 'trust'  :: cod will automatically learn detected help command;
#   - 'ignore' :: cod will ignore all command for this executable.

# Examples:
#   [[rule]]
#   executable = "/usr/bin/*"
#   policy = 'ignore'
# 
#   [[rule]]
#   executable = "~/bin/*"
#   policy = 'trust'

#   [[rule]]
#   executable = "~/my/repo/**"
#   policy = 'trust'
`
