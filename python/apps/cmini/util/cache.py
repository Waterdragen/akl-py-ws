import json
import multiprocessing
import os
import time
from importlib import import_module
from typing import TypeAlias

CorpusName: TypeAlias = str
MetricName: TypeAlias = str

LOADED_TRIGRAMS: dict[str, dict] = {}

if __name__ == "__main__":
    # cd ./apps
    # run python -m cmini.util.cache
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    analyzer = import_module(".analyzer", package="cmini.util")
    memory = import_module(".memory", package="cmini.util")
    resource = import_module(".resource", package="cmini.core")
    keyboard = import_module(".keyboard", package="cmini.core")
    Layout = keyboard.Layout
    AbsPath = resource.AbsPath
else:
    from ..util import analyzer, memory
    from ..core.resource import AbsPath
    from ..core.keyboard import Layout


def timing(func):
    def wrapper(*args, **kwargs):
        start_t = time.perf_counter()
        func(*args, **kwargs)
        end_t = time.perf_counter()
        print(f"Time elapsed: {end_t - start_t:.2f}s")
    return wrapper


def cache_get(name: str) -> dict | None:
    name = name.lower()
    abs_dir = os.path.dirname(__file__)
    path = os.path.join(abs_dir, f'../cache/{name}.json')
    if not os.path.exists(path):
        return None

    with AbsPath(__file__).open(f'../cache/{name}.json', 'r') as f:
        return json.load(f)


def get(name: str, corpus: str):
    name = name.lower()
    corpus = corpus.lower()

    if not name or not corpus:
        return None

    if (data := cache_get(name)) is not None:
        if corpus in data:
            # print("Returning cached data")
            return data[corpus]


def load_trigrams(corpus: str) -> dict:
    if corpus in LOADED_TRIGRAMS:
        return LOADED_TRIGRAMS[corpus]
    with AbsPath(__file__).open(f'../corpora/{corpus}/trigrams.json', 'r') as f:
        d: dict = json.load(f)
        LOADED_TRIGRAMS[corpus] = d
        return d

def get_corpus_cache(layout: Layout, corpus: str) -> dict[MetricName, float]:
    trigrams = load_trigrams(corpus)
    stats = analyzer.trigrams(layout, trigrams)
    return stats

def write_layout_cache(layout_cache: dict, layout_name: str):
    with AbsPath(__file__).open(f"../cache/{layout_name}.json", 'w') as f:
        json.dump(layout_cache, f)

def mp_process_layout(layout_file: str, corpus_names: list[str]):
    layout_name = AbsPath.get_basename(layout_file)
    layout_cache: dict[CorpusName, dict[MetricName, float]] = {}
    layout: Layout = memory.parse_file(layout_file)

    for corpus in corpus_names:
        print(f"Layout: {layout_name}, Corpus: {corpus}")

        layout_cache[corpus] = get_corpus_cache(layout, corpus)

    write_layout_cache(layout_cache, layout_name)


@timing
def mp_main():
    layout_files: list[str] = AbsPath(__file__).glob("../layouts/*")
    corpus_names: list[str] = [AbsPath.get_basename(x) for x in os.listdir("../corpora")]

    num_processes = multiprocessing.cpu_count()

    with multiprocessing.Pool(processes=num_processes) as pool:
        results = []
        for layout_file in layout_files:
            result = pool.apply_async(mp_process_layout, args=(layout_file, corpus_names))
            results.append(result)

        # Wait for all processes to finish
        for result in results:
            result.get()


if __name__ == "__main__":
    mp_main()
