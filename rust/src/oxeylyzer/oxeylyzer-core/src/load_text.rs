use crate::translation::Translator;

use std::fs::{read_dir, File};
use std::iter::FromIterator;
use std::path::PathBuf;
use std::time::Instant;

use anyhow::Result;
use file_chunker::FileChunker;
use fxhash::FxHashMap as HashMap;
use indexmap::IndexMap;
use rayon::iter::{IntoParallelRefIterator, ParallelBridge, ParallelIterator};
use serde::{Deserialize, Serialize};
use smartstring::{LazyCompact, SmartString, SmartStringMode};

use std::sync::{Arc, Mutex};
use actix_ws::Session;
use oxeylyzer_ws::sender::Sendable;

const FOUR_MB: u64 = 1024 * 1024 * 4;

pub struct LoadText {
    session: Arc<Mutex<Session>>,
}

impl Sendable for LoadText {
    fn session(&self) -> &Arc<Mutex<Session>> {
        &self.session
    }
}

impl LoadText {
    pub fn new(session: Arc<Mutex<Session>>) -> Self {
        LoadText {session}
    }

    pub fn load_raw(&self, language: &str) {
        self.load_data(language, Translator::raw(true)).unwrap();
    }

    pub fn load_data(&self, language: &str, translator: Translator) -> Result<()> {
        let start_total = Instant::now();
        let is_raw = translator.is_raw;

        let chunkers = read_dir(format!("static/text/{language}"))?
            .par_bridge()
            .filter_map(Result::ok)
            .flat_map(|dir_entry| File::open(dir_entry.path()))
            .map(|f| {
                let len = f.metadata().unwrap().len() + 1;
                let count = (len / FOUR_MB).max(1);
                (FileChunker::new(&f).unwrap(), count as usize)
            })
            .collect::<Vec<_>>();

        let chunkers_time = Instant::now();
        self.sendln(format!(
            "Prepared text files in {}ms",
            (chunkers_time - start_total).as_millis()
        ));

        let strings = chunkers
            .par_iter()
            .flat_map(|(chunker, count)| chunker.chunks(*count, Some(' ')).unwrap())
            .map(|chunk| {
                std::str::from_utf8(chunk).expect(
                    "one of the files provided is not encoded as utf-8.\
                    Make sure all files in the directory are valid utf-8.",
                )
            })
            .map(|s| {
                let mut last_chars = SmartString::<LazyCompact>::new();
                let mut inter = [' '; 5];
                s.chars()
                    .rev()
                    .take(5)
                    .enumerate()
                    .for_each(|(i, c)| unsafe { *inter.get_unchecked_mut(4 - i) = c });

                inter.into_iter().for_each(|c| last_chars.push(c));
                last_chars.push_str("     ");

                (s, last_chars)
            })
            .collect::<Vec<_>>();

        self.sendln(format!(
            "Converted to utf8 in {}ms",
            (Instant::now() - chunkers_time).as_millis()
        ));

        let quingrams = strings
            .par_iter()
            .map(|(s, last)| TextNgrams::from_str_last(s, &last))
            .reduce(
                || TextNgrams::default(),
                |accum, new| accum.combine_with(new),
            );

        TextData::from((quingrams, language, translator)).save(is_raw)?;
        self.sendln(format!(
            "loading {} took {}ms",
            language,
            (Instant::now() - start_total).as_millis()
        ));

        Ok(())
    }

}


#[derive(Default, Debug)]
pub struct TextNgrams<'a, const N: usize> {
    pub ngrams: HashMap<&'a str, usize>,
}

impl<'a, const N: usize> TextNgrams<'a, N> {
    fn from_str_last<M: SmartStringMode>(s: &'a str, last: &'a SmartString<M>) -> Self {
        let mut ngrams = HashMap::default();
        let it1 = s.char_indices().map(|(i, _)| i);
        let it2 = s.char_indices().map(|(i, _)| i).skip(N);
        it1.zip(it2).map(|(i1, i2)| &s[i1..i2]).for_each(|ngram| {
            ngrams.entry(ngram).and_modify(|f| *f += 1).or_insert(1);
        });

        let it1 = last.char_indices().map(|(i, _)| i);
        let it2 = last.char_indices().map(|(i, _)| i).skip(N);
        it1.zip(it2)
            .map(|(i1, i2)| &last[i1..i2])
            .for_each(|ngram| {
                ngrams.entry(ngram).and_modify(|f| *f += 1).or_insert(1);
            });

        Self { ngrams }
    }
}

impl<'a, const N: usize> TextNgrams<'a, N> {
    fn combine_with(mut self, rhs: Self) -> Self {
        for (trigram, freq) in rhs.ngrams.into_iter() {
            self.ngrams
                .entry(trigram)
                .and_modify(|f| *f += freq)
                .or_insert(freq);
        }
        self
    }
}

#[derive(Default, Debug, Serialize, Deserialize)]
pub struct TextData {
    language: String,

    characters: IndexMap<char, f64>,
    bigrams: IndexMap<SmartString<LazyCompact>, f64>,
    skipgrams: IndexMap<SmartString<LazyCompact>, f64>,
    skipgrams2: IndexMap<SmartString<LazyCompact>, f64>,
    skipgrams3: IndexMap<SmartString<LazyCompact>, f64>,
    trigrams: IndexMap<SmartString<LazyCompact>, f64>,

    #[serde(skip)]
    char_sum: f64,
    #[serde(skip)]
    bigram_sum: f64,
    #[serde(skip)]
    skipgram_sum: f64,
    #[serde(skip)]
    skipgram2_sum: f64,
    #[serde(skip)]
    skipgram3_sum: f64,
    #[serde(skip)]
    trigram_sum: f64,
}

impl std::fmt::Display for TextData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "{{\
                \"language\": {},\
                \"characters\": {:#?},\
                \"bigrams\": {:#?},\
                \"skipgrams\": {:#?},\
                \"skipgrams2\": {:#?},\
                \"skipgrams3\": {:#?},\
                \"trigrams\": {:#?}\
            }}",
            self.language,
            self.characters,
            self.bigrams,
            self.skipgrams,
            self.skipgrams2,
            self.skipgrams3,
            self.trigrams
        )
    }
}

impl TextData {
    pub fn new(language: &str) -> Self {
        let mut res = Self::default();
        res.language = language.replace(" ", "_").to_lowercase().to_string();
        res
    }
}

impl<'a> From<(TextNgrams<'a, 5>, &str, Translator)> for TextData {
    fn from((ngrams, language, translator): (TextNgrams<5>, &str, Translator)) -> Self {
        let mut res = TextData::new(language);

        for (ngram, freq) in ngrams.ngrams.into_iter() {
            let first = unsafe { ngram.chars().next().unwrap_unchecked() };
            if first != ' ' {
                if let Some(first_t) = translator.table.get(&first) {
                    if first_t != " " {
                        let mut trans = translator.translate(ngram);
                        match trans.chars().count() {
                            5.. => {
                                trans.push(' ');
        
                                let first_t_len = first_t.chars().count().max(1);
                                let it1 = trans.char_indices().map(|(i, _)| i).take(first_t_len);
                                let it2 = trans.char_indices().map(|(i, _)| i).skip(5).take(first_t_len);
        
                                it1.zip(it2)
                                    .map(|(i1, i2)| &trans[i1..i2])
                                    .for_each(|ngram| res.from_n_subsequent::<5>(ngram, freq as f64)
                                );
                            }
                            4 => {
                                println!("4 long ngram: '{}'", &trans);
                                res.from_n_subsequent::<4>(&trans, freq as f64)
                            },
                            3 => res.from_n_subsequent::<3>(&trans, freq as f64),
                            2 => res.from_n_subsequent::<2>(&trans, freq as f64),
                            1 => res.from_n_subsequent::<1>(&trans, freq as f64),
                            _ => {}
                        }
                    }
                }
            }
        }

        res.characters
            .iter_mut()
            .for_each(|(_, f)| *f /= res.char_sum);
        res.bigrams
            .iter_mut()
            .for_each(|(_, f)| *f /= res.bigram_sum);
        res.skipgrams
            .iter_mut()
            .for_each(|(_, f)| *f /= res.skipgram_sum);
        res.skipgrams2
            .iter_mut()
            .for_each(|(_, f)| *f /= res.skipgram2_sum);
        res.skipgrams3
            .iter_mut()
            .for_each(|(_, f)| *f /= res.skipgram3_sum);
        res.trigrams
            .iter_mut()
            .for_each(|(_, f)| *f /= res.trigram_sum);

        res.characters
            .sort_by(|_, f1, _, f2| f2.partial_cmp(f1).unwrap());
        res.bigrams
            .sort_by(|_, f1, _, f2| f2.partial_cmp(f1).unwrap());
        res.skipgrams
            .sort_by(|_, f1, _, f2| f2.partial_cmp(f1).unwrap());
        res.skipgrams2
            .sort_by(|_, f1, _, f2| f2.partial_cmp(f1).unwrap());
        res.skipgrams3
            .sort_by(|_, f1, _, f2| f2.partial_cmp(f1).unwrap());
        res.trigrams
            .sort_by(|_, f1, _, f2| f2.partial_cmp(f1).unwrap());

        res
    }
}

impl TextData {
    fn from_n_subsequent<const N: usize>(&mut self, ngram: &str, freq: f64) {
        let mut chars = ngram.chars();
        match chars.next() {
            Some(c1) if N > 0 && c1 != ' ' => {
                self.add_character(c1, freq);
                // take first, first 2 etc chars of the trigram every time for the appropriate stat
                // as long as they don't contain spaces. return `c2` so I don't iter.next() too much
                let c2 = match chars.next() {
                    Some(c2) if N > 1 && c2 != ' ' => {
                        self.add_bigram([c1, c2], freq);
                        c2
                    },
                    _ => ' '
                };

                // c1 and c3 for skipgrams
                match chars.next() {
                    Some(c3) if N > 2 && c3 != ' ' => {
                        self.add_skipgram([c1, c3], freq);

                        if c2 != ' ' {
                            self.add_trigram([c1, c2, c3], freq);
                        }

                        match chars.next() {
                            Some(c4) if N > 3 && c4 != ' ' => {
                                self.add_skipgram2([c1, c4], freq);

                                match chars.next() {
                                    Some(c5) if N > 4 && c5 != ' ' => {
                                        self.add_skipgram3([c1, c5], freq);
                                    },
                                    _ => {}
                                }
                            },
                            _ => {}
                        }
                    },
                    _ => {}
                }
            },
            _ => {},
        }
    }

    pub(crate) fn add_character(&mut self, c: char, freq: f64) {
        self.characters
            .entry(c)
            .and_modify(|e| *e += freq)
            .or_insert(freq);
        self.char_sum += freq;
    }

    pub(crate) fn add_bigram(&mut self, bigram: [char; 2], freq: f64) {
        self.bigrams
            .entry(SmartString::from_iter(bigram))
            .and_modify(|e| *e += freq)
            .or_insert(freq);
        self.bigram_sum += freq;
    }

    pub(crate) fn add_skipgram(&mut self, skipgram: [char; 2], freq: f64) {
        self.skipgrams
            .entry(SmartString::from_iter(skipgram))
            .and_modify(|e| *e += freq)
            .or_insert(freq);
        self.skipgram_sum += freq;
    }

    pub(crate) fn add_skipgram2(&mut self, skipgram: [char; 2], freq: f64) {
        self.skipgrams2
            .entry(SmartString::from_iter(skipgram))
            .and_modify(|e| *e += freq)
            .or_insert(freq);
        self.skipgram2_sum += freq;
    }

    pub(crate) fn add_skipgram3(&mut self, skipgram: [char; 2], freq: f64) {
        self.skipgrams3
            .entry(SmartString::from_iter(skipgram))
            .and_modify(|e| *e += freq)
            .or_insert(freq);
        self.skipgram3_sum += freq;
    }

    pub(crate) fn add_trigram(&mut self, trigram: [char; 3], freq: f64) {
        self.trigrams
            .entry(SmartString::from_iter(trigram))
            .and_modify(|e| *e += freq)
            .or_insert(freq);
        self.trigram_sum += freq;
    }

    fn save(&self, pass: bool) -> Result<()> {
        use std::fs::OpenOptions;
        use std::io::Write;

        let buf = Vec::new();
        let formatter = serde_json::ser::PrettyFormatter::with_indent(b"\t");
        let mut ser = serde_json::Serializer::with_formatter(buf, formatter);
        self.serialize(&mut ser).unwrap();

        let data_dir_str = format!("static/language_data{}", if pass { "_raw" } else { "" });
        let data_dir = &PathBuf::from(data_dir_str);

        if let Ok(true) = data_dir.try_exists() {
            std::fs::create_dir_all(&data_dir)?;
        }

        let mut file = OpenOptions::new()
            .write(true)
            .create(true)
            .truncate(true)
            .open(format!("{}/{}.json", data_dir.to_str().expect("the provided path should be valid utf8"), self.language))?;

        file.write(ser.into_inner().as_slice())?;
        Ok(())
    }
}
