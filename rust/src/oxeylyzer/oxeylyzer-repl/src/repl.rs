use std::io::Write;
use std::path::Path;

use getargs::Options;
use indexmap::IndexMap;
use itertools::Itertools;
use oxeylyzer_core::{generate::LayoutGeneration, layout::*, weights::Config};

use crate::commands::{ArgumentType, Commands};
use crate::tui::TUI;
use ArgumentType::*;

use std::sync::{Arc, Mutex};
use actix_ws::Session;
use oxeylyzer_ws::sender::Sendable;

pub struct UserData {
    language: String,
    gen: LayoutGeneration,
    saved: IndexMap<String, FastLayout>,
    temp_generated: Vec<FastLayout>,
    pins: Vec<usize>,
}

pub struct Repl {
    user_data: UserData,
    session: Arc<Mutex<Session>>,
}

impl Sendable for Repl {
    fn session(&self) -> &Arc<Mutex<Session>> {
        &self.session
    }
}

impl Repl {
    fn new<P>(generator_base_path: P, session: Arc<Mutex<Session>>, user_data: Option<UserData>) -> Result<Self, String>
    where
        P: AsRef<Path>,
    {
        if let Some(user_data) = user_data {
            return Ok(Self{user_data, session});
        }

        let config = Config::new();
        let language = config.defaults.language.clone();
        let pins = config.pins.clone();

        let mut gen = LayoutGeneration::new(
            config.defaults.language.clone().as_str(),
            generator_base_path.as_ref(),
            Some(config),
            Arc::clone(&session),
        )
        .expect(format!("Could not read language data for {}", language).as_str());

        Ok(
            Self {
            user_data: UserData {
                saved: gen
                        .load_layouts(
                            generator_base_path.as_ref().join("layouts"),
                            language.as_str(),
                        )
                        .map_err(|e| e.to_string())?,
                language,
                gen,
                temp_generated: Vec::new(),
                pins,
                },
            session: Arc::clone(&session),
            }
        )
    }

    pub fn run(command: String, session: Arc<Mutex<Session>>, user_data: Option<UserData>) -> Result<Self, String> {
        let inst = Self::new("static", Arc::clone(&session), user_data);
        if let Err(err) = inst {
            return Err(format!("New instance failed, msg: {}", err));
        }
        let mut env = inst.unwrap();

        let wants_exit = env.post_run(command);

        if wants_exit {
            let inst = Self::new("static", Arc::clone(&session), None);
            if let Err(err) = inst {
                return Err(format!("New instance failed, msg: {}", err));
            }
            env = inst.unwrap();
        }

        Ok(env)
    }

    fn post_run(&mut self, command: String) -> bool {
        let line = command.clone();
        let line = line.trim();
        if line.is_empty() {
            self.send("[DONE]");
            return false;
        }
        match self.respond(line) {
            Ok(wants_exit) => {
                if wants_exit {
                    return true;
                }
            },
            Err(err) => {
                self.send(format!("{err}"));
            }
        }
        self.send("[DONE]");
        false
    }

    pub fn get_user_data(self) -> UserData {
        self.user_data
    }

    fn rank(&self) {
        for (name, layout) in self.user_data.saved.iter() {
            self.sendln(format!("{:10}{}", format!("{:.3}:", layout.score), name));
        }
    }

    fn layout_by_name(&self, name: &str) -> Option<&FastLayout> {
        self.user_data.saved.get(name)
    }

    fn analyze_name(&self, name: &str) {
        let l = match self.layout_by_name(name) {
            Some(layout) => layout,
            None => {
                self.sendln(format!("layout {} does not exist!", name));
                return;
            }
        };
        self.sendln(format!("{}", name));
        self.analyze(&l);
    }

    fn placeholder_name(&self, layout: &FastLayout) -> Result<String, String> {
        for i in 1..1000usize {
            let new_name_bytes = layout.matrix[10..14]
                .into_iter()
                .map(|b| *b)
                .collect::<Vec<u8>>();
            let mut new_name = self.user_data.gen.data.convert_u8.as_str(new_name_bytes.as_slice());

            new_name.push_str(format!("{}", i).as_str());

            if !self.user_data.saved.contains_key(&new_name) {
                return Ok(new_name);
            }
        }
        Err("Could not find a good placeholder name for the layout.".to_string())
    }

    #[allow(dead_code)]
    fn save(&mut self, mut layout: FastLayout, name: Option<String>) -> Result<(), String> {
        let new_name = if let Some(n) = name {
            n.replace(" ", "_")
        } else {
            self.placeholder_name(&layout).unwrap()
        };

        let mut f = std::fs::OpenOptions::new()
            .write(true)
            .create(true)
            .truncate(true)
            .open(format!("./src/oxeylyzer/static/layouts/{}/{}.kb", self.user_data.language, new_name))
            .map_err(|e| e.to_string())?;

        let layout_formatted = layout.formatted_string(&self.user_data.gen.data.convert_u8);
        self.sendln(format!("saved {}\n{}", new_name, layout_formatted));
        f.write(layout_formatted.as_bytes()).unwrap();

        layout.score = self.user_data.gen.score(&layout);
        self.user_data.saved.insert(new_name, layout);
        self.user_data.saved
            .sort_by(|_, a, _, b| a.score.partial_cmp(&b.score).unwrap());

        Ok(())
    }

    fn analyze(&self, layout: &FastLayout) {
        let stats = self.user_data.gen.get_layout_stats(layout);
        let score = if layout.score == 0.000 {
            self.user_data.gen.score(layout)
        } else {
            layout.score
        };

        let layout_str = TUI::heatmap_string(&self.user_data.gen.data, layout);

        self.sendln(format!("{}\n{}\nScore: {:.3}", layout_str, stats, score));
    }

    fn compare_name(&self, name1: &str, name2: &str) {
        let l1 = match self.layout_by_name(name1) {
            Some(layout) => layout,
            None => {
                self.sendln(format!("layout {} does not exist!", name1));
                return;
            }
        };
        let l2 = match self.layout_by_name(name2) {
            Some(layout) => layout,
            None => {
                self.sendln(format!("layout {} does not exist!", name2));
                return;
            }
        };
        self.sendln(format!("\n{:31}{}", name1, name2));
        let mut layouts_str = String::with_capacity(1400);
        for y in 0..3 {
            for (n, layout) in [l1, l2].into_iter().enumerate() {
                for x in 0..10 {
                    layouts_str.push_str(&format!("{} ", TUI::heatmap_heat(&self.user_data.gen.data, layout.c(x + 10 * y))));
                    if x == 4 {
                        layouts_str.push_str(" ");
                    }
                }
                if n == 0 {
                    layouts_str.push_str("          ");
                }
            }
            layouts_str.push_str("\n");
        }
        self.send(layouts_str);
        let s1 = self.user_data.gen.get_layout_stats(l1);
        let s2 = self.user_data.gen.get_layout_stats(l2);
        let ts1 = s1.trigram_stats;
        let ts2 = s2.trigram_stats;
        self.sendln(format!(
            concat!(
                "Sfb:               {: <11} Sfb:               {:.3}%\n",
                "Dsfb:              {: <11} Dsfb:              {:.3}%\n",
                "Finger Speed:      {: <11} Finger Speed:      {:.3}\n",
                "Scissors           {: <11} Scissors:          {:.3}%\n",
                "Lsbs               {: <11} Lsbs:              {:.3}%\n\n",
                "Inrolls:           {: <11} Inrolls:           {:.2}%\n",
                "Outrolls:          {: <11} Outrolls:          {:.2}%\n",
                "Total Rolls:       {: <11} Total Rolls:       {:.2}%\n",
                "Onehands:          {: <11} Onehands:          {:.3}%\n\n",
                "Alternates:        {: <11} Alternates:        {:.2}%\n",
                "Alternates Sfs:    {: <11} Alternates Sfs:    {:.2}%\n",
                "Total Alternates:  {: <11} Total Alternates:  {:.2}%\n\n",
                "Redirects:         {: <11} Redirects:         {:.3}%\n",
                "Redirects Sfs:     {: <11} Redirects Sfs:     {:.3}%\n",
                "Bad Redirects:     {: <11} Bad Redirects:     {:.3}%\n",
                "Bad Redirects Sfs: {: <11} Bad Redirects Sfs: {:.3}%\n",
                "Total Redirects:   {: <11} Total Redirects:   {:.3}%\n\n",
                "Bad Sfbs:          {: <11} Bad Sfbs:          {:.3}%\n",
                "Sft:               {: <11} Sft:               {:.3}%\n\n",
                "Score:             {: <11} Score:             {:.3}\n"
            ),
            format!("{:.3}%", s1.sfb * 100.0),
            s2.sfb * 100.0,
            format!("{:.3}%", s1.dsfb * 100.0),
            s2.dsfb * 100.0,
            format!("{:.3}", s1.fspeed * 10.0),
            s2.fspeed * 10.0,
            format!("{:.3}%", s1.scissors * 100.0),
            s2.scissors * 100.0,
            format!("{:.3}%", s1.lsbs * 100.0),
            s2.lsbs * 100.0,
            format!("{:.2}%", ts1.inrolls * 100.0),
            ts2.inrolls * 100.0,
            format!("{:.2}%", ts1.outrolls * 100.0),
            ts2.outrolls * 100.0,
            format!("{:.2}%", (ts1.inrolls + ts1.outrolls) * 100.0),
            (ts2.inrolls + ts2.outrolls) * 100.0,
            format!("{:.3}%", ts1.onehands * 100.0),
            ts2.onehands * 100.0,
            format!("{:.2}%", ts1.alternates * 100.0),
            ts2.alternates * 100.0,
            format!("{:.2}%", ts1.alternates_sfs * 100.0),
            ts2.alternates_sfs * 100.0,
            format!("{:.2}%", (ts1.alternates + ts1.alternates_sfs) * 100.0),
            (ts2.alternates + ts2.alternates_sfs) * 100.0,
            format!("{:.3}%", ts1.redirects * 100.0),
            ts2.redirects * 100.0,
            format!("{:.3}%", ts1.redirects_sfs * 100.0),
            ts2.redirects_sfs * 100.0,
            format!("{:.3}%", ts1.bad_redirects * 100.0),
            ts2.bad_redirects * 100.0,
            format!("{:.3}%", ts1.bad_redirects_sfs * 100.0),
            ts2.bad_redirects_sfs * 100.0,
            format!(
                "{:.3}%",
                (ts1.redirects + ts1.redirects_sfs + ts1.bad_redirects + ts1.bad_redirects_sfs)
                    * 100.0
            ),
            (ts2.redirects + ts2.redirects_sfs + ts2.bad_redirects + ts2.bad_redirects_sfs) * 100.0,
            format!("{:.3}%", ts1.bad_sfbs * 100.0),
            ts2.bad_sfbs * 100.0,
            format!("{:.3}%", ts1.sfts * 100.0),
            ts2.sfts * 100.0,
            format!("{:.3}", l1.score),
            l2.score
        ));
    }

    fn get_nth(&self, nr: usize) -> Option<FastLayout> {
        if nr < self.user_data.temp_generated.len() {
            let l = self.user_data.temp_generated[nr].clone();
            Some(l)
        } else {
            if self.user_data.temp_generated.len() == 0 {
                self.sendln("You haven't generated any layouts yet!");
            } else {
                self.sendln("That's not a valid index!");
            }
            None
        }
    }

    fn sfr_freq(&self) -> f64 {
        let len = self.user_data.gen.data.characters.len();
        let chars = 0..len;
        chars
            .clone()
            .cartesian_product(chars)
            .filter(|(i1, i2)| i1 == i2)
            .map(|(c1, c2)| self.user_data.gen.data.bigrams.get(c1 * len + c2).unwrap_or(&0.0))
            .sum()
    }

    fn sfbs(&self, name: &str, top_n: usize) {
        if let Some(layout) = self.layout_by_name(name) {
            self.sendln(format!("top {} sfbs for {name}:", top_n.min(48)));

            for (bigram, freq) in self.user_data.gen.sfbs(layout, top_n) {
                self.sendln(format!("{bigram}: {:.3}%", freq * 100.0))
            }
        } else {
            self.sendln(format!("layout {name} does not exist!"))
        }
    }

    fn load_language(&mut self, language: &str, config: Config) {
        if let Ok(generator) = LayoutGeneration::new(
            language,
            "static",
            Some(config),
            self.new_session(),
        ) {
            let user_data = &mut self.user_data;
            user_data.language = language.to_string();
            user_data.gen = generator;
            user_data.saved = user_data.gen.load_layouts(
                "static/layouts",
                language
            ).expect("couldn't load layouts lol");

            self.sendln(format!(
                "Set language to {}. Sfr: {:.2}%",
                language, self.sfr_freq() * 100.0
            ));
        } else {
            self.sendln(format!("Could not load data for {language}"));
        }
    }

    fn respond(&mut self, line: &str) -> Result<bool, String> {
        let args = shlex::split(line).ok_or("error: Invalid quoting")?;
        let mut args = Options::new(args.iter().map(String::as_str));

        match args.next_positional() {
            Some("generate") | Some("gen") | Some("g") => {
                if let Some(count_str) = args.next_positional() {
                    if let Ok(count) = usize::from_str_radix(count_str, 10) {
                        if count > 500 {
                            self.sendln(String::from("Cannot generate more than 500 layouts in demo mode"));
                            return Ok(true);
                        }

                        self.sendln(format!("generating {} layouts...", count_str));
                        self.user_data.temp_generated = TUI::new(self.new_session()).generate_n(&self.user_data.gen, count);
                    } else {
                        Commands::new(self.new_session()).send_error("generate", &[R("amount")]);
                    }
                }
            }
            Some("improve") | Some("i") => {
                if let Some(name) = args.next_positional() {
                    if let Some(amount_str) = args.next_positional() {
                        if let Ok(amount) = usize::from_str_radix(amount_str, 10) {
                            if amount > 500 {
                                self.sendln(String::from("Cannot generate more than 500 layouts in demo mode"));
                                return Ok(true);
                            }

                            if let Some(l) = self.layout_by_name(name) {
                                self.user_data.temp_generated = TUI::new(self.new_session()).generate_n_with_pins(&self.user_data.gen, amount, l.clone(), &self.user_data.pins);
                            } else {
                                self.sendln(format!("'{name}' does not exist!"))
                            }
                        } else {
                            Commands::new(self.new_session()).send_error("improve", &[R("name"), R("amount")]);
                        }
                    }
                }
            }
            Some("rank") => self.rank(),
            Some("analyze") | Some("layout") | Some("a") => {
                if let Some(name_or_nr) = args.next_positional() {
                    if let Ok(nr) = usize::from_str_radix(name_or_nr, 10) {
                        if let Some(layout) = self.get_nth(nr) {
                            self.analyze(&layout);
                        }
                    } else {
                        self.analyze_name(name_or_nr);
                    }
                } else {
                    Commands::new(self.new_session()).send_error("analyze", &[R("name or number")]);
                }
            }
            Some("compare") | Some("c") | Some("comp") | Some("cmopare") | Some("comprae") => {
                if let Some(layout1) = args.next_positional() {
                    if let Some(layout2) = args.next_positional() {
                        self.compare_name(layout1, layout2);
                    } else {
                        Commands::new(self.new_session()).send_error("compare", &[R("layout 1"), R("layout 2")]);
                    }
                }
            }
            Some("sfbs") | Some("sfb") => {
                if let Some(name) = args.next_positional() {
                    if let Some(top_n_str) = args.next_positional() {
                        if let Ok(top_n) = usize::from_str_radix(top_n_str, 10) {
                            self.sfbs(name, top_n)
                        } else {
                            Commands::new(self.new_session()).send_error("ngram", &[R("name"), O("top n")]);
                        }
                    } else {
                        self.sfbs(name, 10);
                    }
                } else {
                    Commands::new(self.new_session()).send_error("ngram", &[R("name"), O("top n")]);
                }
            }
            Some("ngram") | Some("occ") | Some("n") => {
                if let Some(ngram) = args.next_positional() {
                    let s = format!("{}", TUI::get_ngram_info(&mut self.user_data.gen.data, ngram));
                    self.sendln(s);
                } else {
                    Commands::new(self.new_session()).send_error("ngram", &[R("ngram")]);
                }
            }
            Some("load") => {
                self.sendln("Unsupported feature in demo mode");
                return Ok(false)
            }
            Some("language") | Some("lanugage") | Some("langauge") | Some("lang") | Some("l") => {
                match args.next_positional() {
                    Some(language) => {
                        let config = Config::new();
                        self.load_language(language, config);
                    }
                    None => self.sendln(format!("Current language: {}", self.user_data.language))
                }
            }
            Some("languages") | Some("langs") => {
                std::fs::read_dir("./src/oxeylyzer/static/language_data")
                    .unwrap()
                    .flatten()
                    .map(|p| p
                        .file_name()
                        .to_string_lossy()
                        .replace("_", " ")
                        .replace(".json", "")
                    )
                    .filter(|n| n != "test")
                    .for_each(|n| self.sendln(format!("{n}")))
            }
            Some("reload") | Some("r") => {
                let config = Config::new();
                self.user_data.pins = config.pins.clone();

                if let Ok(generator) = LayoutGeneration::new(
                    self.user_data.language.as_str(),
                    "static",
                    Some(config),
                    self.new_session(),
                ) {
                    self.user_data.gen = generator;
                    self.user_data.saved = self.user_data.gen.load_layouts(
                        "static/layouts",
                        self.user_data.language.as_str()
                    ).expect("couldn't load layouts lol");
                } else {
                    self.sendln(format!("Could not load {}", self.user_data.language));
                }
            }
            Some("save") | Some("s") => {
                self.sendln("Unsupported feature in demo mode");
                return Ok(false)
            }
            Some("quit") | Some("exit") | Some("q") => {
                self.sendln("Exiting analyzer...");
                self.send("[QUIT]");
                return Ok(true)
            }
            Some("help") | Some("--help") | Some("h") | Some("-h") => {
                match args.next_positional() {
                    Some("generate") | Some("gen") | Some("g") => {
                        Commands::new(self.new_session()).send_help(
                            "generate", 
                            "(g, gen) Generate a number of layouts and shows the best 10, All layouts generated are accessible until reloading or quiting.",
                            &[R("amount")]
                        )
                    }
                    Some("improve") | Some("i") => {
                        Commands::new(self.new_session()).send_help(
                            "improve",
                            "(i) Save the top <number> result that was generated.",
                            &[R("name"), R("amount")]
                        )
                    }
                    Some("rank") => {
                        Commands::new(self.new_session()).send_help(
                            "rank",
                            "(sort) Rank all layouts in set language by score using values set from 'config.toml'",
                            &[]
                        )
                    }
                    Some("analyze") | Some("layout") | Some("a") => {
                        Commands::new(self.new_session()).send_help(
                            "analyze",
                            "(a, layout) Show details of layout.",
                            &[R("name or number")]
                        )
                    }
                    Some("compare") | Some("c") | Some("cmp") | Some("cmopare") | Some("comprae") => {
                        Commands::new(self.new_session()).send_help(
                            "compare",
                            "(c, cmp) Compare 2 layouts.",
                            &[R("layout 1"), R("layout 2")]
                        )
                    }
                    Some("sfbs") | Some("sfb") => {
                        Commands::new(self.new_session()).send_help(
                            "sfbs",
                            "(sfbs, sfb) Shows the top n sfbs for a certain layout.",
                            &[R("name"), O("top n")]
                        )
                    }
                    Some("ngram") | Some("occ") | Some("n") => {
                        Commands::new(self.new_session()).send_help(
                            "ngram",
                            "(n, occ) Gives information about a certain ngram. for 2 letter ones, skipgram info will be provided as well.",
                            &[R("ngram")]
                        )
                    }
                    Some("load") => {
                        Commands::new(self.new_session()).send_help(
                            "load",
                            "Generates corpus for <language>. Will be include everything but spaces if the language is not known.",
                            &[R("language"), O("preferred_config_folder"), A("raw")]
                        )
                    }
                    Some("language") | Some("lanugage") | Some("langauge") | Some("lang") | Some("l") => {
                        Commands::new(self.new_session()).send_help(
                            "language",
                            "(l, lang) Sets a language to be used for analysis.",
                            &[R("language")]
                        )
                    }
                    Some("languages") | Some("langs") => {
                        Commands::new(self.new_session()).send_help(
                            "languages",
                            "(langs) Shows available languages.",
                            &[]
                        )
                    }
                    Some("reload") | Some("r") => {
                        Commands::new(self.new_session()).send_help(
                            "reload",
                            "(r) Reloads all data with the current language. Loses temporary layouts.",
                            &[]
                        )
                    }
                    Some("save") | Some("s") => {
                        Commands::new(self.new_session()).send_help(
                            "save",
                            "(s) Saves the top <number> result that was generated. Starts from 0 up to the number generated.",
                            &[R("index"), O("name")]
                        )
                    }
                    Some("quit") | Some("exit") | Some("q") => {
                        Commands::new(self.new_session()).send_help(
                            "quit",
                            "(q) Quit the repl",
                            &[]
                        )
                    }
                    Some("help") | Some("--help") | Some("h") | Some("-h") => {
                        Commands::new(self.new_session()).send_help(
                            "help",
                            "Print this message or the help of the given subcommand(s)",
                            &[O("subcommand")]
                        )
                    }
                    Some(c) => self.sendln(format!("error: the subcommand '{c}' wasn't recognized")),
                    None => {
                        self.sendln(format!(concat!(
                            "commands:\n",
                            "    analyze      (a, layout) Show details of layout\n",
                            "    compare      (c, comp) Compare 2 layouts\n",
                            "    generate     (g, gen) Generate a number of layouts and shows the best 10, All layouts\n",
                            "                     generated are accessible until reloading or quiting.\n",
                            "    help         Print this message or the help of the given subcommand(s)\n",
                            "    improve      (i, optimize) Save the top <NR> result that was generated. Starts from 1, Takes\n",
                            "                     negative values\n",
                            "    language     (l, lang) Set a language to be used for analysis. Loads corpus when not present\n",
                            "    languages    (langs) Show available languages\n",
                            "    load         Generates corpus for <language>. Will be exclude spaces from source if the\n",
                            "                     language isn't known\n",
                            "    ngram        (occ) Gives information about a certain ngram. for 2 letter ones, skipgram info\n",
                            "                     will be provided as well.\n",
                            "    quit         (q) Quit the repl\n",
                            "    rank         (sort) Rank all layouts in set language by score using values set from\n",
                            "                     'config.toml'\n",
                            "    reload       (r) Reloads all data with the current language. Loses temporary layouts.\n",
                            "    save         (s) Save the top <NR> result that was generated. Starts from 1 up to the number\n",
                            "                     generated, Takes negative values\n"
                        )));
                    }
                }
            }
            Some(c) => self.sendln(format!("error: the command '{c}' wasn't recognized")),
            None => {}
        }

        Ok(false)
    }
}
