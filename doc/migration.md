# Migration depuis la 1.x

La version 2 est une réécriture complète en Go de l'ancien `process-monitor` en Python ; le format de config a changé.

## Correspondance des options {#options}

| 1.x | 2.x |
|---|---|
| `process-monitor.exe --config x.yaml` | `pm run x.yaml` (lancer une fois) ou `pm watch x.yaml` (relancer) |
| `process_name` | `track` (ou rien si c'est le programme lancé) |
| `executable` | `command` |
| `working_dir` | `workdir` |
| `monitor_process: false` / `true` | `pm run` / `pm watch` |
| `kill_existing: true` | `if_running: kill` |
| `lock_file` | `ready.file`, avec un délai |
| `initial_file_copies` + `final_file_copies` en miroir | une ligne `sync` par fichier |
| `initial_file_cleanup` / `final_file_cleanup` | `hooks.before` / `hooks.after` avec `delete` |
| `capture_output` + `output_log` | `capture` + `log` |
| `log_cleanup_*` | `log.keep` |
| `exit_delay`, `wait_keypress` | `exit.delay`, `exit.wait_key` |
| `interval`, `startup_delay` | supprimés : pm ne sonde plus, il attend la fin du process |
| `tui`, `verbose` | `pm watch` a l'interface, `pm run` la console ; tout va au journal |

## Comportements changés {#changes}

- `q` ne tue plus le programme : il détache, après confirmation. `k` arrête.
- Les actions de fin s'exécutent une seule fois (la 1.x les jouait jusqu'à trois fois).
- Un programme déjà lancé n'est plus « adopté » : pm refuse (ou l'arrête avec `if_running: kill`).
- Une action *avant* en échec empêche le lancement (la 1.x le journalisait et lançait quand même).
- `exit.wait_key` s'applique aussi à l'interface de `pm watch` (la 1.x ignorait `wait_keypress` en mode TUI).
- La copie retour attend la fin du jeu et de son lanceur, pas celle des autres process qu'il a lancés (la 1.x n'attendait que le jeu).
- En `pm watch`, chaque relance rejoue les actions *avant* mais pas les copies `sync` ; la 1.x recopiait aussi ses `initial_file_copies` à chaque relance, ce qui écrasait les réglages modifiés pendant le lancement précédent.
