import { Link } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { ScrollView, StyleSheet, Text } from 'react-native';

import { useAuth } from '@/lib/auth';
import { MinimalHeader } from '@/components/MinimalHeader';
import { colors, layout, spacing, type } from '@/lib/theme';

const licenseText = `Copyright (c) 2020, OpenTaal
All rights reserved.

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`;

const iconoirLicense = `MIT License

Copyright (c) 2021 Daniel Martin, Pascal Jufer

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`;

export default function DisclaimerScreen() {
  const { user } = useAuth();

  return (
    <SafeAreaView style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.container} showsVerticalScrollIndicator={false}>
        <MinimalHeader backHref={user?.profileComplete ? '/home' : user ? '/onboarding' : '/login'} title="Data and licenses" />

        <Text style={styles.title}>Translations and Dutch definitions</Text>
        <Text style={styles.body}>
          Our 9,500-word collection uses Wiktionary contributors’ translations, collected through DBnary and WikDict by Karl Bartel. Dictionary version 2025.11.21. Woordenboom selects translations and Dutch definitions for matching meanings, adds editorial explanations and corrections, and orders words using frequency estimates.
        </Text>
        <Text style={styles.body}>
          The dictionary and this adapted collection are available under Creative Commons Attribution-ShareAlike 3.0. Translations and definitions are drafts awaiting language review. Some explanations were written with AI assistance. Examples are available for part of the collection.
        </Text>
        <Link href="https://www.wikdict.com/page/download" style={styles.detail}>Dictionary source and downloads</Link>
        <Link href="https://nl.wiktionary.org/" style={styles.detail}>Wiktionary contributors</Link>
        <Link href="https://creativecommons.org/licenses/by-sa/3.0/" style={styles.detail}>Read the CC BY-SA 3.0 license</Link>
        <Text style={styles.title}>Frequency estimates</Text>
        <Text style={styles.body}>Word order uses wordfreq by Robyn Speer and contributors, version 3.0.2. Its code uses Apache-2.0; its data includes attributed and share-alike sources.</Text>
        <Link href="https://github.com/rspeer/wordfreq/blob/master/NOTICE.md" style={styles.detail}>wordfreq source notices</Link>
        <Text style={styles.title}>OpenTaal attribution</Text>
        <Text style={styles.body}>
          Woordenboom includes the &quot;Dutch word list&quot; by OpenTaal. We use version 2.20.23 from commit b250510dda431785f962019167d1415198ff3905 under the Revised BSD License.
        </Text>
        <Text style={styles.detail}>Creator: OpenTaal, https://opentaal.org</Text>
        <Text style={styles.detail}>Source: https://github.com/OpenTaal/opentaal-wordlist</Text>
        <Text style={styles.body}>OpenTaal and its contributors do not endorse Woordenboom.</Text>
        <Text style={styles.license}>{licenseText}</Text>
        <Text style={styles.title}>Iconoir interface icons</Text>
        <Link href="https://iconoir.com" style={styles.detail}>Iconoir project</Link>
        <Text style={styles.license}>{iconoirLicense}</Text>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: layout.screen, container: layout.page, title: type.title, body: type.body,
  detail: type.caption,
  license: { ...type.caption, color: colors.inkSoft, borderTopWidth: 1, borderTopColor: colors.line, paddingTop: spacing.xl },
});
