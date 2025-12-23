from typing import Literal, Iterable

import argparse
import csv
import os
from collections import defaultdict

import matplotlib.pyplot as plt


GROUPS_DICT_TYPE = dict[Literal["algo"], list[str]]


def read_results_csv(path: str):
    """
    Считывает CSV-файл с результатами бенчмарка алгоритмов
    """
    rows = []
    with open(path, "r", encoding="utf-8") as f:
        r = csv.DictReader(f)
        for row in r:
            rows.append({
                "algo": row["algo"],  # Название алгоритма
                "jobs": int(row["jobs"]),  # Количество работ
                "machines": int(row["machines"]),  # Количество станков
                "runs": int(row["runs"]),  # Число прогонов
                "time_mean_ms": float(row["time_mean_ms"]),  # Среднее время выполнения (мс)
                "time_std_ms": float(row["time_std_ms"]),  # Стандартное отклонение времени выполнения (мс)
                "makespan_best": int(row["makespan_best"]),  # Лучшее найденное значение makespan
                "makespan_mean": float(row["makespan_mean"]),  # Среднее значение makespan по прогонам
                "makespan_std": float(row["makespan_std"]),  # Стандартное отклонение makespan
            })
    return rows


def group_by_algo(rows: Iterable) -> GROUPS_DICT_TYPE:
    """
    Группирует строки результатов по алгоритмам.

    Возвращает словарь:
        algo -> список строк,
    где внутри каждой группы данные отсортированы
    по (jobs, machines) для корректного построения графиков.
    """
    g = defaultdict(list)
    for row in rows:
        g[row["algo"]].append(row)

    # Сортировка внутри каждого алгоритма
    for algo in g:
        g[algo].sort(key=lambda x: (x["jobs"], x["machines"]))

    return g


def plot_lines(groups: GROUPS_DICT_TYPE, out_path: str, title: str, y_label: str, y_key: str):
    """
    Строит линейный график зависимости метрики y_key
    от числа работ (jobs) для всех алгоритмов.

    Используется для:
      - best makespan
      - mean makespan
      - std makespan
    """
    fig = plt.figure()
    ax = fig.add_subplot(111)

    # Каждая линия — отдельный алгоритм
    for algo, rows in sorted(groups.items(), key=lambda kv: kv[0]):
        x = [r["jobs"] for r in rows]
        y = [r[y_key] for r in rows]
        ax.plot(x, y, marker="o", linestyle="-", label=algo)

    ax.set_xlabel("n_jobs")
    ax.set_ylabel(y_label)
    ax.set_title(title)

    # Сетка для удобства визуального анализа
    ax.grid(True, which="both", linestyle=":", linewidth=0.7)
    ax.legend()

    fig.tight_layout()
    fig.savefig(out_path, dpi=170)
    plt.close(fig)


def plot_runtime_mean_with_std(groups: GROUPS_DICT_TYPE, out_path: str):
    """
    Строит график среднего времени выполнения
    с отображением стандартного отклонения (error bars).

    Используется для анализа производительности алгоритмов.
    """
    fig = plt.figure()
    ax = fig.add_subplot(111)

    for algo, rows in sorted(groups.items(), key=lambda kv: kv[0]):
        x = [r["jobs"] for r in rows]
        y = [r["time_mean_ms"] for r in rows]
        yerr = [r["time_std_ms"] for r in rows]

        # errorbar показывает mean ± std
        ax.errorbar(
            x,
            y,
            yerr=yerr,
            marker="o",
            linestyle="-",
            capsize=3,
            label=algo,
        )

    ax.set_xlabel("n_jobs")
    ax.set_ylabel("runtime mean (ms)")
    ax.set_title("Runtime vs jobs (mean ± std)")
    ax.grid(True, which="both", linestyle=":", linewidth=0.7)
    ax.legend()

    fig.tight_layout()
    fig.savefig(out_path, dpi=170)
    plt.close(fig)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument(
        "--in",
        dest="inp",
        default="artifacts/results.csv",
        help="путь к входному CSV-файлу с результатами",
    )
    ap.add_argument(
        "--outdir",
        dest="outdir",
        default="artifacts",
        help="директория для сохранения графиков",
    )
    args = ap.parse_args()

    rows = read_results_csv(args.inp)
    if not rows:
        raise SystemExit("CSV пуст или не удалось распарсить данные")

    # На всякий случай создаём выходную директорию
    os.makedirs(args.outdir, exist_ok=True)

    groups = group_by_algo(rows)

    # 1) Производительность (время выполнения)
    p1 = os.path.join(args.outdir, "runtime_mean_ms.png")
    plot_runtime_mean_with_std(groups, p1)

    # 2) Лучшее значение целевой функции
    p2 = os.path.join(args.outdir, "makespan_best.png")
    plot_lines(
        groups,
        p2,
        "Makespan best vs jobs",
        "makespan best",
        "makespan_best",
    )

    # 3) Среднее значение целевой функции
    p3 = os.path.join(args.outdir, "makespan_mean.png")
    plot_lines(
        groups,
        p3,
        "Makespan mean vs jobs",
        "makespan mean",
        "makespan_mean",
    )

    # 4) Стандартное отклонение целевой функции
    p4 = os.path.join(args.outdir, "makespan_std.png")
    plot_lines(
        groups,
        p4,
        "Makespan std vs jobs",
        "makespan std",
        "makespan_std",
    )

    print("Сохранено:")
    print(" -", p1)
    print(" -", p2)
    print(" -", p3)
    print(" -", p4)


if __name__ == "__main__":
    main()
