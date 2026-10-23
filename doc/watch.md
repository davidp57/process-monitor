# pm watch

```powershell
.\pm.exe watch <config.yaml> [--profile NOM] [--tray | --no-tui] [--icon FICHIER] [--lang fr|en]
```

`pm watch` relance le programme à chaque arrêt, selon `watch.restart` : `always` (défaut), `on-failure` (seulement sur un code de sortie non nul) ou `never`.

## Relances {#restarts}

- Un arrêt moins de 10 secondes après le lancement compte comme un plantage en boucle : le premier attend `watch.delay`, chaque suivant double le délai, jusqu'à `watch.max_delay`.
- `watch.max_restarts` ne compte que les relances automatiques, pas celles demandées avec `r`.
- Les lignes `sync` sont copiées une fois au début et recopiées une fois à la fin de la session.
- Les autres actions *avant* (un nettoyage, par exemple) sont rejouées avant chaque relance ; les actions *après* ne s'exécutent qu'à la fin.
- À chaque fin de lancement, pm arrête ce qui reste du groupe de process, pour que les restes ne s'empilent pas d'un lancement à l'autre.

## Touches {#keys}

| Touche | Effet |
|---|---|
| `p` | met en pause / reprend le programme (tous ses process) |
| `r` | arrête et relance le programme |
| `k` | arrête le programme, exécute les actions *après*, quitte |
| `q` deux fois | détache : pm quitte, le programme continue, les actions *après* ne sont **pas** exécutées ; au démarrage, annule ; pendant le délai avant une relance, termine la session, actions *après* comprises |
| flèches, PgUp/PgDn | font défiler le journal |

`--no-tui` remplace l'interface par un journal en texte simple, par exemple pour un lancement en service.
`--tray` la remplace par l'[icône de notification](tray.md).
