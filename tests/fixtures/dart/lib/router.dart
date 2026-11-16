import 'package:flutter/foundation.dart';
import 'package:go_router/go_router.dart';

import 'screens/about_screen.dart';
import 'screens/debug_screen.dart';
import 'screens/help_screen.dart';
import 'screens/home_screen.dart';
import 'screens/login_screen.dart';
import 'screens/order_screen.dart';
import 'screens/wireframe_gallery.dart';

final appRouter = GoRouter(
  initialLocation: '/',
  routes: [
    GoRoute(path: '/', builder: (context, state) => const HomeScreen()),
    GoRoute(path: '/login', builder: (context, state) => const LoginScreen()),
    GoRoute(
      path: '/orders/:id',
      builder: (context, state) => OrderScreen(id: state.pathParameters['id']!),
    ),
    GoRoute(
      path: '/help',
      builder: (context, state) =>
          HelpScreen(url: state.uri.queryParameters['page'] ?? 'https://paywallet.example/help'),
    ),
    GoRoute(path: '/about', builder: (context, state) => const AboutScreen()),
    GoRoute(path: '/debug', builder: (context, state) => const DebugScreen()),
    if (kDebugMode)
      GoRoute(path: '/wireframes', builder: (context, state) => const WireframeGallery()),
  ],
);
