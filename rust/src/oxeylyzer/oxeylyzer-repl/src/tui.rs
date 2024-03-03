use oxeylyzer_core::generate::LayoutGeneration;
use oxeylyzer_core::language_data::LanguageData;
use oxeylyzer_core::layout::*;
use oxeylyzer_core::rayon::iter::ParallelIterator;

use ansi_rgb::{rgb, Colorable};
use indicatif::ProgressBar;

use std::sync::{Arc, Mutex};
use actix_ws::Session;
use oxeylyzer_ws::sender::Sendable;
use std::time::Duration;

pub(crate) struct TUI {
    session: Arc<Mutex<Session>>,
}

impl Sendable for TUI {
    fn session(&self) -> &Arc<Mutex<Session>> {
        &self.session
    }
}

impl TUI {
    pub fn new(session: Arc<Mutex<Session>>) -> Self {
        TUI {session}
    }

    pub fn heatmap_heat(data: &LanguageData, c: u8) -> String {
        let complement = 215.0 - *data.characters.get(c as usize).unwrap_or_else(|| &0.0) * 1720.0;
        let complement = complement.max(0.0) as u8;
        let heat = rgb(215, complement, complement);
        let c = data.convert_u8.from_single(c);
        format!("{}", c.to_string().fg(heat))
    }

    pub fn heatmap_string(data: &LanguageData, layout: &FastLayout) -> String {
        let mut print_str = String::new();

        for (i, c) in layout.matrix.iter().enumerate() {
            if i % 10 == 0 && i > 0 {
                print_str.push('\n');
            }
            if (i + 5) % 10 == 0 {
                print_str.push(' ');
            }
            print_str.push_str(Self::heatmap_heat(data, *c).as_str());
            print_str.push(' ');
        }

        print_str
    }

    pub fn generate_n_with_pins(
        &self,
        gen: &LayoutGeneration,
        amount: usize,
        based_on: FastLayout,
        pins: &[usize],
    ) -> Vec<FastLayout> {
        if amount == 0 {
            return Vec::new();
        }

        let start = std::time::Instant::now();

        let pb = ProgressBar::hidden();
        pb.set_length(amount as u64);

        let mut layouts = gen
            .generate_n_with_pins_iter(amount, based_on, pins)
            .map(|value| {
                self.sendln(format!("[PROGRESS]{}", generate_progress_bar(&pb)));
                value
            })
            .collect::<Vec<_>>();

        self.sendln(format!(
            "[PROGRESS]Optimizing {} variants took: {} seconds",
            amount,
            start.elapsed().as_secs()
        ));

        layouts.sort_by(|l1, l2| l2.score.partial_cmp(&l1.score).unwrap());

        for (i, layout) in layouts.iter().enumerate().take(10) {
            let printable = Self::heatmap_string(&gen.data, layout);
            self.sendln(format!("#{}, score: {:.5}\n{}", i, layout.score, printable));
        }

        layouts
    }

    pub fn generate_n(&self, gen: &LayoutGeneration, amount: usize) -> Vec<FastLayout> {
        if amount == 0 {
            return Vec::new();
        }

        let start = std::time::Instant::now();

        let pb = ProgressBar::hidden();
        pb.set_length(amount as u64);

        let mut layouts = gen
            .generate_n_iter(amount)
            .map(|value| {
                self.sendln(format!("[PROGRESS]{}", generate_progress_bar(&pb)));
                value
            })
            .collect::<Vec<_>>();

        self.sendln(format!(
            "[PROGRESS]optimizing {} variants took: {} seconds",
            amount,
            start.elapsed().as_secs()
        ));

        layouts.sort_by(|l1, l2| l2.score.partial_cmp(&l1.score).unwrap());

        for (i, layout) in layouts.iter().enumerate().take(10) {
            let printable = Self::heatmap_string(&gen.data, layout);
            self.sendln(format!("#{}, score: {:.5}\n{}", i, layout.score, printable));
        }

        layouts
    }

    pub fn get_ngram_info(data: &mut LanguageData, ngram: &str) -> String {
        match ngram.chars().count() {
            1 => {
                let c = ngram.chars().next().unwrap();
                let u = data.convert_u8.to_single(c);
                let occ = data.characters.get(u as usize).unwrap_or(&0.0) * 100.0;
                format!("{ngram}: {occ:.3}%")
            }
            2 => {
                let bigram: [char; 2] = ngram.chars().collect::<Vec<char>>().try_into().unwrap();
                let c1 = data.convert_u8.to_single(bigram[0]) as usize;
                let c2 = data.convert_u8.to_single(bigram[1]) as usize;

                let b1 = c1 * data.characters.len() + c2;
                let b2 = c2 * data.characters.len() + c1;

                let rev = bigram.into_iter().rev().collect::<String>();

                let occ_b1 = data.bigrams.get(b1).unwrap_or(&0.0) * 100.0;
                let occ_b2 = data.bigrams.get(b2).unwrap_or(&0.0) * 100.0;
                let occ_s = data.skipgrams.get(b1).unwrap_or(&0.0) * 100.0;
                let occ_s2 = data.skipgrams.get(b2).unwrap_or(&0.0) * 100.0;

                format!(
                    "{ngram} + {rev}: {:.3}%,\n  {ngram}: {occ_b1:.3}%\n  {rev}: {occ_b2:.3}%\n\
                {ngram} + {rev} (skipgram): {:.3}%,\n  {ngram}: {occ_s:.3}%\n  {rev}: {occ_s2:.3}%",
                    occ_b1 + occ_b2,
                    occ_s + occ_s2
                )
            }
            3 => {
                let trigram: [char; 3] = ngram.chars().collect::<Vec<char>>().try_into().unwrap();
                let t = [
                    data.convert_u8.to_single(trigram[0]),
                    data.convert_u8.to_single(trigram[1]),
                    data.convert_u8.to_single(trigram[2]),
                ];
                let &(_, occ) = data
                    .trigrams
                    .iter()
                    .find(|&&(tf, _)| tf == t)
                    .unwrap_or(&(t, 0.0));
                format!("{ngram}: {:.3}%", occ * 100.0)
            }
            _ => "Invalid ngram! It must be 1, 2 or 3 chars long.".to_string(),
        }
    }

}

fn format_duration(duration: Duration) -> String {
    let seconds = duration.as_secs();
    let hours = seconds / 3600;
    let minutes = (seconds % 3600) / 60;
    let seconds = seconds % 60;

    format!("{:02}:{:02}:{:02}", hours, minutes, seconds)
}

fn generate_progress_bar(pb: &ProgressBar) -> String {
    pb.inc(1);

    let elapsed = pb.elapsed();
    let position = pb.position();
    let eta = pb.eta();
    let total_len = pb.length().unwrap();
    let per_sec = pb.per_sec();

    let bar_length = 70;
    let progress = position as f32 / total_len as f32;
    let completed_length = (bar_length as f32 * progress) as u32;
    let todo_length = bar_length - completed_length;

    let completed_bar = format!("{}>", "=".repeat(completed_length as usize));
    let todo_bar = "-".repeat(todo_length as usize);

    let formatted_elapsed = format_duration(elapsed);
    let eta_secs = eta.as_secs();

    format!("[{}] [{}{}] [eta: {:>3}s] - {:>11}/s {:>6}/{}", formatted_elapsed, completed_bar, todo_bar, eta_secs, format!("{:.2}", per_sec), position, total_len)
}


