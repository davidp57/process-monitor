# pm 2.1.0

**pm s'utilise maintenant sans terminal ni fichier à écrire : un double-clic sur `pm.exe` ouvre son interface, un assistant crée la config et ses raccourcis.**

---

## Un double-clic, une interface

Un double-clic sur `pm.exe` ouvre l'interface de pm dans votre navigateur, sans rien à installer :

- **Sessions** : tout ce que pm fait tourner, d'où que ce soit parti (un raccourci, un terminal, l'interface), avec le journal en direct ; on peut arrêter, mettre en pause, relancer ou détacher, comme depuis l'icône de notification.
- **Configs** : vos configs, retrouvées toutes seules à côté de `pm.exe`, dans `Saved Games\DCS\Profiles` et là où l'assistant les range. Pour chacune : la lancer avec le profil choisi, voir en un dessin quels fichiers du profil vont où avant et après la partie, lire ses journaux, créer ses raccourcis, et la **modifier par formulaires**, sans toucher au YAML.
- **Nouvelle config** : l'assistant, dans la page.
- Un bouton **Parcourir…** à côté de chaque chemin, pour le choisir dans la fenêtre habituelle de Windows plutôt que de le taper.
- La **langue** se règle dans l'interface, pour tout pm, même si Windows est dans une autre langue ; thème clair ou sombre au choix.

L'interface se ferme toute seule une demi-minute après le dernier onglet fermé ; les sessions en cours, elles, continuent.

## Un assistant pour créer une config

`pm new` (ou « Nouvelle config » dans l'interface) pose quelques questions et écrit une config qui marche, avec ses raccourcis : DCS par profils, un autre jeu à profils, un programme à garder en vie, ou une sauvegarde avant de lancer un programme.

Pour un jeu à profils, **chaque profil se règle en jouant** : l'assistant lance le jeu, vous réglez ce que vous voulez, vous quittez, et il montre les fichiers qui ont changé pour que vous gardiez les bons.
Les réglages d'origine sont remis entre chaque profil et à la fin, et rien n'est écrit avant votre confirmation.
Une config existante n'est jamais écrasée.

## Des raccourcis tout faits

`pm shortcut` crée un raccourci Windows par profil, sur le Bureau, dans le menu Démarrer ou à côté de la config, avec l'icône du profil.
Un raccourci qui existait déjà est gardé en `.lnk.old`.
Pour qui préfère le terminal, `pm menu` présente tout ce que pm sait faire, à choisir dans une liste.

## Corrigé

- Arrêter un jeu depuis l'icône de notification n'affiche plus d'erreur.

## À vérifier dans votre config

`desktop` et `start_menu` sont maintenant des variables fournies par pm (le Bureau et le menu Démarrer).
Si l'une de vos configs définit elle-même une variable de ce nom dans `vars:`, pm la refuse : renommez-la. `pm check` vous le dira.

---

📖 [Documentation](https://davidp57.github.io/process-monitor/latest/) · [L'interface](https://davidp57.github.io/process-monitor/latest/gui/) · [L'assistant](https://davidp57.github.io/process-monitor/latest/new/) · 🇬🇧 [English](https://davidp57.github.io/process-monitor/latest/en/)
